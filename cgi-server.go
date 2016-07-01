package main

import (
	"net/http"
	"net/http/cgi"
	"fmt"
)

func main() {

	fmt.Println("starting server on http://localhost:9001")
	
	api := cgi.Handler{}
	api.Path = "aceapi-v1"

	updater := cgi.Handler{}
	updater.Path = "updater"

	mux := http.NewServeMux()
	mux.Handle("/v1/", &api)
	mux.Handle("/updater/", &updater)

	err := http.ListenAndServeTLS(":9001", "server.pem", "server.key", mux)
	if err != nil {
		panic(err)
	}
}
