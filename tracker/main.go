package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/ShreyamKundu/peernet/tracker/api"
	"github.com/ShreyamKundu/peernet/tracker/config"
	"github.com/ShreyamKundu/peernet/tracker/db"
	"github.com/ShreyamKundu/peernet/tracker/reputation"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.New()

	database, err := db.InitDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Start the reputation engine
	reputationEngine := reputation.NewEngine(database)
	go reputationEngine.Start()

	// Set up Gin router
	router := setupRouter(database, cfg.JWTSecret)
	
	// Set up the HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reputationEngine.Stop() // Stop the reputation engine
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func setupRouter(database *sql.DB, jwtSecret string) *gin.Engine {
	router := gin.Default()
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	apiV1 := router.Group("/api/v1")
	api.RegisterRoutes(apiV1, database, jwtSecret)

	return router
}