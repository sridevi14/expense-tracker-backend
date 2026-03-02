package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Data interface{} `json:"data"`
}

type ListResponse struct {
	Data       interface{} `json:"data"`
	NextCursor *string     `json:"next_cursor,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, SuccessResponse{Data: data})
}

func List(c *gin.Context, data interface{}, nextCursor *string) {
	c.JSON(http.StatusOK, ListResponse{Data: data, NextCursor: nextCursor})
}

func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{Code: code, Message: message},
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", message)
}

func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

func ValidationError(c *gin.Context, message string) {
	Error(c, http.StatusUnprocessableEntity, "VALIDATION_REQUIRED", message)
}
