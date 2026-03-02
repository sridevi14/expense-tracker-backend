package model

import "time"

type Budget struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Month     time.Time `json:"month"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
