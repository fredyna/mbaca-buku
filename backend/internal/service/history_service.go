package service

import (
	"context"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/repository"
)

type HistoryService struct {
	historyRepo *repository.HistoryRepository
}

func NewHistoryService(historyRepo *repository.HistoryRepository) *HistoryService {
	return &HistoryService{historyRepo: historyRepo}
}

func (s *HistoryService) GetHistory(ctx context.Context, userID string) (*dto.HistoryResponse, error) {
	items, err := s.historyRepo.GetUserHistory(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := &dto.HistoryResponse{
		Reading:   []dto.HistoryItem{},
		Completed: []dto.HistoryItem{},
	}

	for _, item := range items {
		if item.Status == "completed" {
			resp.Completed = append(resp.Completed, item)
		} else {
			resp.Reading = append(resp.Reading, item)
		}
	}

	return resp, nil
}
