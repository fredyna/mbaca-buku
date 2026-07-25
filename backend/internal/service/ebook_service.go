package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/repository"
	"github.com/fredy/mbaca-buku/internal/storage"
)

type EbookService struct {
	ebookRepo *repository.EbookRepository
	storage   *storage.MinIOStorage
}

func NewEbookService(ebookRepo *repository.EbookRepository, storage *storage.MinIOStorage) *EbookService {
	return &EbookService{ebookRepo: ebookRepo, storage: storage}
}

func (s *EbookService) Upload(ctx context.Context, req dto.EbookUploadRequest, file io.Reader, fileSize int64, fileName string, userID string) (*model.Ebook, error) {
	objectName := fmt.Sprintf("ebooks/%s/%s", userID, fileName)
	key, err := s.storage.UploadFile(ctx, objectName, file, fileSize, "application/pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	ebook := &model.Ebook{
		Title:      req.Title,
		Author:     req.Author,
		FileURL:    key,
		FileSize:   fileSize,
		TotalPages: req.TotalPages,
		UploadedBy: userID,
	}

	if err := s.ebookRepo.Create(ctx, ebook); err != nil {
		return nil, fmt.Errorf("failed to save ebook: %w", err)
	}

	return ebook, nil
}

func (s *EbookService) GetByID(ctx context.Context, id string) (*model.Ebook, error) {
	return s.ebookRepo.GetByID(ctx, id)
}

func (s *EbookService) List(ctx context.Context, page, perPage int) ([]*model.Ebook, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 50 {
		perPage = 20
	}
	return s.ebookRepo.List(ctx, page, perPage)
}

func (s *EbookService) Update(ctx context.Context, id string, req dto.EbookUpdateRequest) (*model.Ebook, error) {
	ebook, err := s.ebookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	ebook.Title = req.Title
	ebook.Author = req.Author

	if err := s.ebookRepo.Update(ctx, ebook); err != nil {
		return nil, err
	}

	return s.ebookRepo.GetByID(ctx, id)
}

func (s *EbookService) Delete(ctx context.Context, id string) error {
	ebook, err := s.ebookRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	if ebook.FileURL != "" {
		_ = s.storage.DeleteFile(ctx, ebook.FileURL)
	}
	if ebook.CoverURL != "" {
		_ = s.storage.DeleteFile(ctx, ebook.CoverURL)
	}

	return nil
}

func (s *EbookService) GetFileURL(ctx context.Context, id string) (string, error) {
	ebook, err := s.ebookRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	return s.storage.GetPresignedURL(ctx, ebook.FileURL, 1*time.Hour)
}
