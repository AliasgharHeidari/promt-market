package service

import (
	"context"

	"promt-market/internal/repository"
)

type PromptViewService struct {
	repo *repository.PromptViewRepository
}

func NewPromptViewService(repo *repository.PromptViewRepository) *PromptViewService {
	return &PromptViewService{repo: repo}
}

func (s *PromptViewService) Record(ctx context.Context, promptID string) error {
	return s.repo.IncrementToday(ctx, promptID)
}