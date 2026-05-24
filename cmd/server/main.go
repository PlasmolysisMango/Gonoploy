package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/PlasmolysisMango/Gonopoly/internal/server"
)

func main() {
	port := flag.String("port", ":8080", "server listen address")
	saveDir := flag.String("save-dir", "saves/rooms", "directory for room persistence")
	flag.Parse()

	storage := server.NewFileStorage(*saveDir)
	hub := server.NewHub(storage)
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("Gonoploy server starting on %s", *port)
	if err := http.ListenAndServe(*port, mux); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
