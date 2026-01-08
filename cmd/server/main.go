package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"insightsim/internal/database"
	"insightsim/internal/handlers"
	"insightsim/internal/services"

	"github.com/gorilla/mux"
)

func main() {
	// Parse command line flags
	dbPath := flag.String("db", "insightsim.db", "Path to SQLite database file")
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	// Initialize database
	db, err := database.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Printf("Database initialized at: %s", *dbPath)

	// Initialize services
	loader := services.NewLoader(db)
	queryService := services.NewQueryService(db)
	generator := services.NewGenerator(db)

	// Initialize handlers
	loadHandler := handlers.NewLoadHandler(loader)
	queryHandler := handlers.NewQueryHandler(queryService)
	generatorHandler := handlers.NewGeneratorHandler(generator)

	// Setup router
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/load", loadHandler.Handle).Methods("POST")
	api.HandleFunc("/generate-dummy", generatorHandler.Handle).Methods("POST")
	// Handle timeseriesdata with flexible path matching
	api.PathPrefix("/timeseriesdata/").HandlerFunc(queryHandler.Handle).Methods("GET")

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	}).Methods("GET")

	// Start server
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Server starting on port %s", *port)
	log.Printf("API endpoints:")
	log.Printf("  POST /api/load")
	log.Printf("  POST /api/generate-dummy")
	log.Printf("  GET  /api/timeseriesdata/{start}/{end}?tags=<tag1,tag2>")
	log.Printf("  GET  /health")

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
