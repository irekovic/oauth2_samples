package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"oauth2primer/demoutil"
	"os"
	"path"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/oklog/run"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

var bind = flag.String("bind", ":8080", "network binding")
var tenant = flag.String("tenant", "29eb35f9-68d5-4214-98d2-a6c0d92d5a75", "tennant")
var clientID = flag.String("clientid", "11e9dc8f-6e52-47a4-992c-456014ee98b5", "Your client ID")
var clientSecret = flag.String("clientsecret", "4iD8IhR-_q4W-y4_k45Z3dNA9Hy9OV01I2", "Your client secret")

// var issuerURL = flag.String("issuer", "https://login.microsoftonline.com/29eb35f9-68d5-4214-98d2-a6c0d92d5a75/v2.0", "Issuer of the token we will verify!")

var issuerURL = flag.String("issuer", "https://sts.windows.net/29eb35f9-68d5-4214-98d2-a6c0d92d5a75/", "Issuer of the token we will verify!")

var store *sessions.FilesystemStore

// var store *sqlitestore.SqliteStore

// var store = sessions.NewCookieStore([]byte("never_do_this_on_production"))

func main() {
	var err error
	flag.Parse()
	wd, _ := os.Getwd()

	store = sessions.NewFilesystemStore(path.Join(wd, "sessions"), []byte("never_do_this_on_production"))
	store.MaxLength(math.MaxInt64)

	conf := &oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		Scopes: []string{
			oidc.ScopeOpenID,
			oidc.ScopeOfflineAccess,
			"api://11e9dc8f-6e52-47a4-992c-456014ee98b5/webapi",
		},
		RedirectURL: "http://localhost:8080/callback",
		Endpoint:    microsoft.AzureADEndpoint(*tenant),
	}

	provider, err := oidc.NewProvider(context.Background(), *issuerURL)

	if err != nil {
		log.Fatalf("Unable to construct oidc provider: %s\n", err.Error())
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: "api://" + *clientID, SkipIssuerCheck: false})
	setupRoutes(conf, verifier)

	var g run.Group
	g.Add(demoutil.RunSignalHandler())
	g.Add(demoutil.RunHTTPServer(*bind))
	log.Println(g.Run())
}

func setupRoutes(conf *oauth2.Config, verifier *oidc.IDTokenVerifier) {
	r := mux.NewRouter()
	r.StrictSlash(false)
	r.Use(tracingMw)
	r.HandleFunc("/", loginUserMW(homepage(conf))).Methods(http.MethodGet)
	r.HandleFunc("/login", login(conf)).Methods(http.MethodGet)
	r.HandleFunc("/logout", loginUserMW(logout)).Methods(http.MethodGet, http.MethodPost)
	r.HandleFunc("/callback", callback(conf, verifier)).Methods(http.MethodGet)

	http.Handle("/", r)
}

func login(conf *oauth2.Config) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log.Println("Redirecting to ip")
		// just redirect to azure login page
		http.Redirect(rw, r, conf.AuthCodeURL("supersecretrandom"), http.StatusTemporaryRedirect)
	}
}

func callback(conf *oauth2.Config, verifier *oidc.IDTokenVerifier) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		state := r.FormValue("state")
		code := r.FormValue("code")
		log.Println("received callback!", state, code)

		// first check if state is correct! - we are not doing that in PoC
		_ = state

		// do code for token exchange
		token, err := conf.Exchange(r.Context(), code)
		if err != nil {
			log.Println("Unable to obtain token from ad:", err.Error())
			http.Error(rw, err.Error(), http.StatusForbidden)
			return
		}
		log.Println("successfully obtained access token")

		// now that we have token, we should validate it!
		// _, err = verifier.Verify(r.Context(), token.Extra("id_token").(string))
		_, err = verifier.Verify(r.Context(), token.AccessToken)
		if err != nil {
			log.Println("Failed to verifiy access token:", err.Error())
			http.Error(rw, err.Error(), http.StatusForbidden)
			return
		}
		log.Println("verfied token got from exchange")

		// now that we have user's token, we should remember it in session so that we can use
		// it to get access token for other resources that are on different servers.
		session, err := store.New(r, "oauth")
		if err != nil {
			log.Println("Unable to create new session!")
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println("prepared session to save login info")

		session.Values["access"] = token.AccessToken
		session.Values["refresh"] = token.RefreshToken
		session.Values["type"] = token.TokenType
		session.Values["exp"] = token.Expiry.Unix()

		if err := session.Save(r, rw); err != nil {
			log.Println("Error while saving session", err.Error())
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println("Redirecting to homepage! - after login!")
		http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
	}
}

type idtoken struct{}

func loginUserMW(h http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log.Println("loginMW: In login MW")
		session, err := store.Get(r, "oauth")
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("loginMW: found session")
		if session.IsNew {
			log.Println("loginMW: Session is new so redirecting to login page")
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		token := oauth2.Token{
			AccessToken:  session.Values["access"].(string),
			RefreshToken: session.Values["refresh"].(string),
			TokenType:    session.Values["type"].(string),
			Expiry:       time.Unix(session.Values["exp"].(int64), 0),
		}

		// log.Println("loginMW: inflated token", token)
		if !token.Valid() {
			log.Println("loginMW: token is invalid - unauthorized!")
			// http.Error(rw, "Unauthorized", http.StatusForbidden)
			// http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			// return
		}
		// log.Println("storing token(s) in request context: ")
		// log.Println("Access token:", token.AccessToken)
		// log.Println("Refresh token:", token.RefreshToken)
		r = r.WithContext(context.WithValue(r.Context(), idtoken{}, &token))
		h.ServeHTTP(rw, r)
	}
}

func logout(rw http.ResponseWriter, r *http.Request) {
	if session, err := store.Get(r, "oauth"); err == nil {
		delete(session.Values, "access")
		delete(session.Values, "refresh")
		delete(session.Values, "type")
		delete(session.Values, "exp")
		session.Options.MaxAge = -1
		session.Save(r, rw)
		// GET post_logout_redirect_uri=http%3A%2F%2Flocalhost%2Fmyapp%2F
		http.Redirect(rw, r, "https://login.microsoftonline.com/"+*tenant+"/oauth2/v2.0/logout", http.StatusTemporaryRedirect)
	} else {
		log.Println("logout: session not found")
	}
}
func homepage(cfg *oauth2.Config) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		log.Println("Homepage controller")
		token := r.Context().Value(idtoken{}).(*oauth2.Token)

		conf := &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Scopes: []string{
				oidc.ScopeOpenID,
				oidc.ScopeOfflineAccess,
				"api://1c73ba46-367a-493e-b537-06365c20135e",
			},
			// RedirectURL: "http://localhost:8080/callback",
			Endpoint: microsoft.AzureADEndpoint(*tenant),
		}

		cli := conf.Client(r.Context(), token)
		resp, err := cli.PostForm(cfg.Endpoint.TokenURL, url.Values{
			"grant_type":          []string{"urn:ietf:params:oauth:grant-type:jwt-bearer"},
			"client_id":           []string{cfg.ClientID},
			"client_secret":       []string{cfg.ClientSecret},
			"assertion":           []string{token.AccessToken},
			"scope":               []string{"api://1c73ba46-367a-493e-b537-06365c20135e/.default"},
			"requested_token_use": []string{"on_behalf_of"},
		})
		if err != nil {
			// log.Println("Error", err.Error())
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		fmt.Fprintf(rw, "Access Token:\n%v\n", token.AccessToken)
		out, _ := httputil.DumpResponse(resp, true)
		fmt.Fprintf(rw, "OBO Token:\n%v\n", string(out))

	}
	// now obtain token for pricing engine:
}

func tracingMw(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// log.Println("In tracing mw")
		// out, _ := httputil.DumpRequest(r, true)
		// log.Println(string(out))
		h.ServeHTTP(rw, r)
	})
}
