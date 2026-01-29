package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"insightsim/internal/config"
	"insightsim/internal/database"
	"insightsim/internal/handlers"
	"insightsim/internal/services"

	"github.com/gorilla/mux"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.json", "Path to config file")
	dbPath := flag.String("db", "", "Path to SQLite database file (overrides config)")
	port := flag.String("port", "", "Server port (overrides config)")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfigWithDefaults(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override with command line flags if provided
	if *dbPath != "" {
		cfg.Database.Path = *dbPath
	}
	if *port != "" {
		cfg.Server.Port = *port
	}

	// Initialize database
	db, err := database.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Printf("Database initialized at: %s", cfg.Database.Path)

	// Initialize services
	loader := services.NewLoader(db)
	queryService := services.NewQueryService(db)
	generator := services.NewGenerator(db)
	uploadService := services.NewUploadService(db)
	tagsService := services.NewTagsService(db)

	// Get value range from config (with defaults)
	minValue := 1.0
	maxValue := 10000.0
	if cfg.Data.ValueRange != nil {
		minValue = cfg.Data.ValueRange.Min
		maxValue = cfg.Data.ValueRange.Max
	}

	// Get sequential generation flag from config (default: false)
	useSequential := cfg.Data.UseSequentialGeneration

	// Get generation time range from config (with defaults)
	startTime := cfg.Data.GenerationStartTime
	endTime := cfg.Data.GenerationEndTime
	if startTime == "" {
		startTime = "2025-12-01T00:00:00"
	}
	if endTime == "" {
		endTime = "2026-01-31T23:59:59"
	}

	// Initialize handlers with config
	loadHandler := handlers.NewLoadHandler(loader, cfg.Data.RawDataFolder)
	queryHandler := handlers.NewQueryHandler(queryService)
	generatorHandler := handlers.NewGeneratorHandler(generator, minValue, maxValue, useSequential, startTime, endTime)
	uploadHandler := handlers.NewUploadHandler(uploadService)
	tagsHandler := handlers.NewTagsHandler(tagsService)

	// Setup router
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/load", loadHandler.Handle).Methods("POST")
	api.HandleFunc("/generate-dummy", generatorHandler.Handle).Methods("POST")
	api.HandleFunc("/upload-csv", uploadHandler.Handle).Methods("POST")
	api.HandleFunc("/tags/names", tagsHandler.HandleListNames).Methods("GET")
	api.HandleFunc("/tags", tagsHandler.HandleGet).Methods("GET")
	api.HandleFunc("/tags", tagsHandler.HandleDelete).Methods("DELETE")
	api.HandleFunc("/tags", tagsHandler.HandlePost).Methods("POST")
	// Handle timeseriesdata with flexible path matching
	api.PathPrefix("/timeseriesdata/").HandlerFunc(queryHandler.Handle).Methods("GET")

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	}).Methods("GET")

	// CORS middleware: allow all origins in dev
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Start server with CORS wrapper
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	log.Printf("API endpoints:")
	log.Printf("  POST /api/load")
	log.Printf("  POST /api/generate-dummy")
	log.Printf("  POST /api/upload-csv")
	log.Printf("  GET  /api/timeseriesdata/{start}/{end}?tags=<tag1,tag2>")
	log.Printf("  GET  /api/tags")
	log.Printf("  DELETE /api/tags?tag=<name>")
	log.Printf("  GET  /api/tags/names")
	log.Printf("  POST /api/tags")
	log.Printf("  GET  /health")

	if err := http.ListenAndServe(addr, corsMiddleware(router)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
