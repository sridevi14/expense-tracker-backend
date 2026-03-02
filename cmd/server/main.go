package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"

	"expense-tracker/internal/config"
	"expense-tracker/internal/handler"
	"expense-tracker/internal/middleware"
	"expense-tracker/internal/repository"
	"expense-tracker/internal/service"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()

	migDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open migration db connection: %v", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("failed to set goose dialect: %v", err)
	}

	migDB.Close()
	log.Println("migrations complete")

	// ── Connect to PostgreSQL (connection pool) ──────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("connected to PostgreSQL")

	// ── Connect to Redis ─────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}
	defer rdb.Close()
	log.Println("connected to Redis")

	// ── Wire up repositories, services, handlers ─────────────────────────────
	userRepo := repository.NewUserRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	expenseRepo := repository.NewExpenseRepository(pool)
	budgetRepo := repository.NewBudgetRepository(pool)

	authService := service.NewAuthService(userRepo, categoryRepo, cfg.JWTSecret)
	categoryService := service.NewCategoryService(categoryRepo)
	expenseService := service.NewExpenseService(expenseRepo, rdb)
	budgetService := service.NewBudgetService(budgetRepo)

	authHandler := handler.NewAuthHandler(authService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	expenseHandler := handler.NewExpenseHandler(expenseService)
	budgetHandler := handler.NewBudgetHandler(budgetService)

	// ── Gin router ───────────────────────────────────────────────────────────
	r := gin.New()

	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(cfg.CORSOrigin))
	r.Use(middleware.RateLimiter(rdb))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")

	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	protected := api.Group("")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		protected.GET("/categories", categoryHandler.List)
		protected.POST("/categories", categoryHandler.Create)

		protected.GET("/expenses", expenseHandler.List)
		protected.POST("/expenses", expenseHandler.Create)
		protected.PUT("/expenses/:id", expenseHandler.Update)
		protected.DELETE("/expenses/:id", expenseHandler.Delete)
		protected.GET("/expenses/summary", expenseHandler.Summary)

		protected.GET("/budget", budgetHandler.Get)
		protected.PUT("/budget", budgetHandler.Upsert)
	}

	// ── HTTP server with graceful shutdown ────────────────────────────────────
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}
	log.Println("server exited")
}
