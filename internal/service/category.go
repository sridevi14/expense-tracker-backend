package service

import (
	"context"

	"expense-tracker/internal/model"
	"expense-tracker/internal/repository"
)

type CategoryService struct {
	categoryRepo *repository.CategoryRepository
}

func NewCategoryService(categoryRepo *repository.CategoryRepository) *CategoryService {
	return &CategoryService{categoryRepo: categoryRepo}
}

type CreateCategoryInput struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

func (s *CategoryService) Create(ctx context.Context, userID string, input CreateCategoryInput) (*model.Category, error) {
	cat := &model.Category{
		UserID: userID,
		Name:   input.Name,
		Icon:   input.Icon,
	}

	if err := s.categoryRepo.Create(ctx, cat); err != nil {
		return nil, err
	}

	return cat, nil
}

func (s *CategoryService) List(ctx context.Context, userID string) ([]model.Category, error) {
	return s.categoryRepo.ListByUser(ctx, userID)
}

func (s *CategoryService) CreateDefaults(ctx context.Context, userID string) error {
	return s.categoryRepo.CreateDefaults(ctx, userID)
}
