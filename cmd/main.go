package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Just-Goo/Oauth2-Clients/internal/clients"
	_ "github.com/joho/godotenv/autoload"
)

func main() {

	port := os.Getenv("PORT")
	googleClient := clients.NewGoogleClient(port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", googleClient.Index)
	mux.HandleFunc("/login", googleClient.Login)
	mux.HandleFunc("/oauth2/callback", googleClient.Callback)

	server := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Println("server running on port:", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start http server: %v", err)
	}
}
