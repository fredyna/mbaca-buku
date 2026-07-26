package service

import (
	"context"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/repository"
)

type BookmarkService struct {
	bookmarkRepo *repository.BookmarkRepository
}

func NewBookmarkService(bookmarkRepo *repository.BookmarkRepository) *BookmarkService {
	return &BookmarkService{bookmarkRepo: bookmarkRepo}
}

func (s *BookmarkService) Create(ctx context.Context, userID, ebookID string, req dto.CreateBookmarkRequest) (*model.Bookmark, error) {
	bookmark := &model.Bookmark{
		UserID:     userID,
		EbookID:    ebookID,
		PageNumber: req.PageNumber,
		Note:       req.Note,
	}

	if err := s.bookmarkRepo.Create(ctx, bookmark); err != nil {
		return nil, err
	}
	return bookmark, nil
}

func (s *BookmarkService) List(ctx context.Context, userID, ebookID string) ([]model.Bookmark, error) {
	return s.bookmarkRepo.ListByUserAndEbook(ctx, userID, ebookID)
}

func (s *BookmarkService) Delete(ctx context.Context, id, userID string) error {
	return s.bookmarkRepo.Delete(ctx, id, userID)
}
