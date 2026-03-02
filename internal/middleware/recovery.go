package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"expense-tracker/internal/response"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v", err)
				response.InternalError(c, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

func HealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
