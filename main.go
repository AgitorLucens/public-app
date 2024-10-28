package main

import (
	"log"
	"net/http"
)

func main() {
	initOAuth() // Initialize the OAuth configuration

	http.HandleFunc("/oauth/login", loginHandler)
	http.HandleFunc("/oauth/callback", callbackHandler)

	log.Println("Server starting on :3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}