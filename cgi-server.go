package main

import (
	"fmt"
	"net/http"
	"net/http/cgi"
)

func main() {

	fmt.Println("starting server on http://localhost:9001")

	api := cgi.Handler{}
	api.Path = "aceapi"
	api.InheritEnv = []string{"HOME"}

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
