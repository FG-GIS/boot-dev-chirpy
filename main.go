package main

import (
	"log"
	"net/http"
)

func main() {
	port := "8080"
	filepathRoot := "/"
	mux := http.NewServeMux()
	mux.Handle(filepathRoot, http.FileServer(http.Dir(".")))
	server := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
