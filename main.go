package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/alimasry/go-collab-editor/ot"
	"github.com/alimasry/go-collab-editor/server"
	"github.com/alimasry/go-collab-editor/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	memStore := store.NewMemoryStore()
	engine := &ot.JupiterEngine{}
	hub := server.NewHub(memStore, engine)
	go hub.Run()

	handler := server.NewHandler(hub)

	log.Printf("Starting server on %s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
