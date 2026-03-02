package model

import "time"

type Expense struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	CategoryID   string    `json:"category_id"`
	CategoryName string    `json:"category_name,omitempty"`
	Amount       float64   `json:"amount"`
	Description  string    `json:"description"`
	ExpenseDate  time.Time `json:"expense_date"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ExpenseFilter struct {
	UserID     string
	CategoryID string
	StartDate  *time.Time
	EndDate    *time.Time
	Cursor     string
	Limit      int
}

type ExpenseSummary struct {
	TotalAmount float64           `json:"total_amount"`
	Count       int               `json:"count"`
	ByCategory  []CategorySummary `json:"by_category"`
	ByDayOfWeek []DaySummary      `json:"by_day_of_week"`
}

type CategorySummary struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	TotalAmount  float64 `json:"total_amount"`
	Count        int     `json:"count"`
}

type DaySummary struct {
	DayOfWeek   string  `json:"day_of_week"`
	TotalAmount float64 `json:"total_amount"`
	Count       int     `json:"count"`
}
