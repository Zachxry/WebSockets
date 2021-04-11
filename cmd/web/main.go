package main

import (
	"WebSockets/internal/handlers"
	"log"
	"net/http"
)

func main() {
	mux := routes()

	log.Println("Starting Channel Listener")
	go handlers.ListenWsChannel()

	log.Println("Starting Web Server on port 8080")

	_ = http.ListenAndServe(":8080", mux)

}
