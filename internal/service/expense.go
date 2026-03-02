package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"expense-tracker/internal/model"
	"expense-tracker/internal/repository"
)

type ExpenseService struct {
	expenseRepo *repository.ExpenseRepository
	rdb         *redis.Client
}

func NewExpenseService(expenseRepo *repository.ExpenseRepository, rdb *redis.Client) *ExpenseService {
	return &ExpenseService{expenseRepo: expenseRepo, rdb: rdb}
}

type CreateExpenseInput struct {
	CategoryID  string  `json:"category_id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	ExpenseDate string  `json:"expense_date"`
}

type UpdateExpenseInput struct {
	CategoryID  string  `json:"category_id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	ExpenseDate string  `json:"expense_date"`
}

func (s *ExpenseService) Create(ctx context.Context, userID string, input CreateExpenseInput) (*model.Expense, error) {
	expDate, err := time.Parse("2006-01-02", input.ExpenseDate)
	if err != nil {
		expDate = time.Now()
	}

	exp := &model.Expense{
		UserID:      userID,
		CategoryID:  input.CategoryID,
		Amount:      input.Amount,
		Description: input.Description,
		ExpenseDate: expDate,
	}

	if err := s.expenseRepo.Create(ctx, exp); err != nil {
		return nil, err
	}

	s.invalidateSummaryCache(ctx, userID)
	return exp, nil
}

func (s *ExpenseService) Update(ctx context.Context, id, userID string, input UpdateExpenseInput) (*model.Expense, error) {
	expDate, err := time.Parse("2006-01-02", input.ExpenseDate)
	if err != nil {
		expDate = time.Now()
	}

	exp := &model.Expense{
		ID:          id,
		UserID:      userID,
		CategoryID:  input.CategoryID,
		Amount:      input.Amount,
		Description: input.Description,
		ExpenseDate: expDate,
	}

	if err := s.expenseRepo.Update(ctx, exp); err != nil {
		return nil, err
	}

	s.invalidateSummaryCache(ctx, userID)
	return exp, nil
}

func (s *ExpenseService) Delete(ctx context.Context, id, userID string) error {
	if err := s.expenseRepo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.invalidateSummaryCache(ctx, userID)
	return nil
}

func (s *ExpenseService) List(ctx context.Context, filter model.ExpenseFilter) ([]model.Expense, *string, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	filter.Limit = limit

	expenses, err := s.expenseRepo.List(ctx, filter)
	if err != nil {
		return nil, nil, err
	}

	var nextCursor *string
	if len(expenses) > limit {
		cursor := expenses[limit-1].CreatedAt.Format(time.RFC3339Nano)
		nextCursor = &cursor
		expenses = expenses[:limit]
	}

	return expenses, nextCursor, nil
}

func (s *ExpenseService) Summary(ctx context.Context, userID string, startDate, endDate *time.Time) (*model.ExpenseSummary, error) {
	cacheKey := fmt.Sprintf("summary:%s", userID)
	if startDate != nil {
		cacheKey += ":" + startDate.Format("2006-01-02")
	}
	if endDate != nil {
		cacheKey += ":" + endDate.Format("2006-01-02")
	}

	cached, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var summary model.ExpenseSummary
		if json.Unmarshal([]byte(cached), &summary) == nil {
			return &summary, nil
		}
	}

	summary, err := s.expenseRepo.Summary(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(summary); err == nil {
		s.rdb.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return summary, nil
}

func (s *ExpenseService) invalidateSummaryCache(ctx context.Context, userID string) {
	pattern := fmt.Sprintf("summary:%s*", userID)
	iter := s.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		s.rdb.Del(ctx, iter.Val())
	}
}
