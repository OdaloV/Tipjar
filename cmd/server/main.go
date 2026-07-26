package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"tipjar/internal/db"
	"tipjar/internal/handlers"
	"tipjar/internal/repository"
	"tipjar/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading from environment")
	}

	// Connect to database
	db.Connect()

	// Repositories
	jarRepo := repository.NewJarRepository(db.DB)
	txRepo := repository.NewTransactionRepository(db.DB)

	// Services
	jarService := services.NewJarService(jarRepo, txRepo)
	mpesaService := services.NewMpesaService(txRepo, jarRepo)

	// Handlers
	jarHandler := handlers.NewJarHandler(jarService)
	mpesaHandler := handlers.NewMpesaHandler(mpesaService)

	// Routes
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Jar routes
	mux.HandleFunc("/", jarHandler.Home)
	mux.HandleFunc("/jars", jarHandler.CreateJar)
	mux.HandleFunc("/jars/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/jars/":
			http.Redirect(w, r, "/", http.StatusSeeOther)
		case r.URL.Path == "/jars/status":
			jarHandler.GetStatus(w, r)
		case strings.HasSuffix(r.URL.Path, "/pay"):
			mpesaHandler.Pay(w, r)
		default:
			jarHandler.GetJar(w, r)
		}
	})

	// M-Pesa callback
	mux.HandleFunc("/mpesa/callback", mpesaHandler.Callback)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
