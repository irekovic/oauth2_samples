package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http/httputil"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/microsoft"
)

var bind = flag.String("bind", ":9001", "bind address")
var tenant = flag.String("tenant", "29eb35f9-68d5-4214-98d2-a6c0d92d5a75", "tennant")
var clientID = flag.String("clientid", "41744c81-15b1-4987-b87e-23cf885ca738", "Your client ID")
var clientSecret = flag.String("clientsecret", "0sr4bpgytw7i3k21-~T5oU3~n~4_P5mpk.", "Your client secret")
var issuerURL = flag.String("issuer", "https://login.microsoftonline.com/29eb35f9-68d5-4214-98d2-a6c0d92d5a75/v2.0", "Issuer of the token we will verify!")
var sessionKey = flag.String("sessionkey", "randomkey", "Random key used for encripting cookies where we store session")

func main() {
	conf := &clientcredentials.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		Scopes:       []string{"api://1c73ba46-367a-493e-b537-06365c20135e/.default", oidc.ScopeOpenID},
		TokenURL:     microsoft.AzureADEndpoint(*tenant).TokenURL,
	}
	cli := conf.Client(context.Background())

	var priceJson bytes.Buffer
	json.NewEncoder(&priceJson).Encode(map[string]interface{}{"price": time.Now().String()})
	resp, err := cli.Post("http://localhost:9000/", "application/json", &priceJson)
	if err != nil {
		log.Panicln(err)
	}

	out, _ := httputil.DumpResponse(resp, true)
	log.Println("POST RESPONSE")
	log.Println(string(out))

	resp, err = cli.Get("http://localhost:9000/")
	if err != nil {
		log.Panicln(err)
	}

	out, _ = httputil.DumpResponse(resp, true)
	log.Println("GET RESPONSE")
	log.Println(string(out))
}
