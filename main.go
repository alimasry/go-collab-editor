package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/alimasry/go-collab-editor/ot"
	"github.com/alimasry/go-collab-editor/server"
	"github.com/alimasry/go-collab-editor/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	storeType := flag.String("store", "memory", "Storage backend: memory or firestore")
	project := flag.String("project", "", "GCP project ID (required for firestore store)")
	flag.Parse()

	// Cloud Run sets PORT; override -addr if present.
	if port := os.Getenv("PORT"); port != "" {
		*addr = ":" + port
	}

	var docStore store.DocumentStore
	switch *storeType {
	case "memory":
		docStore = store.NewMemoryStore()
	case "firestore":
		projectID := *project
		if projectID == "" {
			projectID = os.Getenv("GCP_PROJECT")
		}
		if projectID == "" {
			log.Fatal("Firestore store requires -project flag or GCP_PROJECT env var")
		}
		client, err := firestore.NewClient(context.Background(), projectID)
		if err != nil {
			log.Fatalf("Failed to create Firestore client: %v", err)
		}
		defer client.Close()
		fsStore := store.NewFirestoreStore(client)
		cachedStore := store.NewCachedStore(fsStore, 5*time.Second)
		defer cachedStore.Close()
		docStore = cachedStore
		log.Printf("Using Firestore store with write-behind cache (project: %s)", projectID)
	default:
		log.Fatalf("Unknown store type: %s", *storeType)
	}

	engine := &ot.JupiterEngine{}
	hub := server.NewHub(docStore, engine)
	go hub.Run()

	handler := server.NewHandler(hub)

	log.Printf("Starting server on %s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
