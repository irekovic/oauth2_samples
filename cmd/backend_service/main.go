package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"oauth2primer/demoutil"
	"oauth2primer/security"
	"sync"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/oklog/run"
)

var bind = flag.String("bind", ":9000", "network binding")
var issuerURL = flag.String("iss", "https://sts.windows.net/29eb35f9-68d5-4214-98d2-a6c0d92d5a75/", "issuer")
var clientID = flag.String("aud", "api://1c73ba46-367a-493e-b537-06365c20135e", "audience")

type state struct {
	price interface{}
	sync.RWMutex
}

var instanceState = state{
	price: map[string]interface{}{
		"sub":   nil,
		"price": "not yet set",
	},
}

func main() {
	var g run.Group
	flag.Parse()
	log.Println("Starting pricing engine daemon")

	// setup routes
	mw, err := security.NewManager(*issuerURL, *clientID)
	if err != nil {
		log.Fatal(err.Error())
	}

	r := mux.NewRouter()
	r.Handle("/", alice.New(mw.AuthorizeMW, security.OnlyInRole("Price.Read")).ThenFunc(readPrice)).Methods(http.MethodGet)
	r.Handle("/", alice.New(mw.AuthorizeMW, security.OnlyInRole("Price.Write")).ThenFunc(writePrice)).Methods(http.MethodPost)
	http.Handle("/", r)

	// setup http server

	// setup runtime dependencies
	g.Add(demoutil.RunSignalHandler())
	g.Add(demoutil.RunHTTPServer(*bind))

	// run everything till completion
	log.Println(g.Run())
}

func readPrice(rw http.ResponseWriter, r *http.Request) {
	instanceState.RLock()
	defer instanceState.RUnlock()

	if err := json.NewEncoder(rw).Encode(instanceState.price); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func writePrice(rw http.ResponseWriter, r *http.Request) {
	instanceState.Lock()
	defer instanceState.Unlock()

	var newState map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&newState); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	token := security.Token(r)
	if token != nil {
		newState["sub"] = token.Subject
	}
	instanceState.price = newState
}
