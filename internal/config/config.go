package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	Port          string
	DatabaseURL   string
	RedisURL      string
	RedisPassword string
	JWTSecret     string
	CORSOrigin    string

	// Raw DB fields needed by entrypoint (exposed for health checks)
	DBHost string
	DBPort string
	DBUser string
	DBName string
}

func Load() (*Config, error) {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "expense_tracker")
	dbSSL := getEnv("DB_SSLMODE", "disable")

	fmt.Println(dbHost, "aa")

	port, err := strconv.Atoi(dbPort)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	// Use url.UserPassword so special characters in credentials are properly escaped.
	dbURL := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbUser, dbPass),
		Host:     fmt.Sprintf("%s:%d", dbHost, port),
		Path:     "/" + dbName,
		RawQuery: "sslmode=" + dbSSL,
	}
	fmt.Println(dbURL, "dburl;")

	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   dbURL.String(),
		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		CORSOrigin:    getEnv("CORS_ORIGIN", "http://localhost:5173"),
		DBHost:        dbHost,
		DBPort:        dbPort,
		DBUser:        dbUser,
		DBName:        dbName,
	}

	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "dev-secret-change-in-production"
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
