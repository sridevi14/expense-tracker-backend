package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"expense-tracker/internal/middleware"
	"expense-tracker/internal/model"
	"expense-tracker/internal/repository"
	"expense-tracker/internal/response"
	"expense-tracker/internal/service"
)

type ExpenseHandler struct {
	expenseService *service.ExpenseService
}

func NewExpenseHandler(expenseService *service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{expenseService: expenseService}
}

func (h *ExpenseHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input service.CreateExpenseInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if input.CategoryID == "" || input.Amount <= 0 {
		response.ValidationError(c, "category_id and a positive amount are required")
		return
	}

	exp, err := h.expenseService.Create(c.Request.Context(), userID, input)
	if err != nil {
		response.InternalError(c, "failed to create expense")
		return
	}

	response.Success(c, http.StatusCreated, exp)
}

func (h *ExpenseHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	var input service.UpdateExpenseInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if input.CategoryID == "" || input.Amount <= 0 {
		response.ValidationError(c, "category_id and a positive amount are required")
		return
	}

	exp, err := h.expenseService.Update(c.Request.Context(), id, userID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c, "expense not found")
			return
		}
		response.InternalError(c, "failed to update expense")
		return
	}

	response.Success(c, http.StatusOK, exp)
}

func (h *ExpenseHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	err := h.expenseService.Delete(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c, "expense not found")
			return
		}
		response.InternalError(c, "failed to delete expense")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *ExpenseHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	filter := model.ExpenseFilter{
		UserID:     userID,
		CategoryID: c.Query("category_id"),
		Cursor:     c.Query("cursor"),
	}

	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}

	if v := c.Query("start_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.StartDate = &t
		}
	}

	if v := c.Query("end_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.EndDate = &t
		}
	}

	expenses, nextCursor, err := h.expenseService.List(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "failed to list expenses")
		return
	}

	if expenses == nil {
		expenses = []model.Expense{}
	}

	response.List(c, expenses, nextCursor)
}

func (h *ExpenseHandler) Summary(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var startDate, endDate *time.Time

	if v := c.Query("start_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			startDate = &t
		}
	}

	if v := c.Query("end_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			endDate = &t
		}
	}

	summary, err := h.expenseService.Summary(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		response.InternalError(c, "failed to get expense summary")
		return
	}

	response.Success(c, http.StatusOK, summary)
}
