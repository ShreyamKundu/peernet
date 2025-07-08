package config

import "os"

// Config holds all configuration for the application.
type Config struct {
	Port string
	DatabaseURL string
	JWTSecret string
}


func New() *Config {
	return &Config {
		Port: getEnv("TRACKER_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://user:password@db:5432/peernet?sslmode=disable"),
		JWTSecret: getEnv("JWT_SECRET", "a_very_secret_key_that_should_be_changed"),
	}
}


func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	} 

	return defaultValue
}
