package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/pdfutil"
	"github.com/fredy/mbaca-buku/internal/repository"
	"github.com/fredy/mbaca-buku/internal/storage"
)

var (
	ErrForbidden  = errors.New("you do not have permission to perform this action")
	ErrNotFound   = errors.New("ebook not found")
	ErrInvalidPDF = errors.New("file is not a readable PDF")
)

type EbookService struct {
	ebookRepo *repository.EbookRepository
	storage   *storage.R2Storage
}

func NewEbookService(ebookRepo *repository.EbookRepository, storage *storage.R2Storage) *EbookService {
	return &EbookService{ebookRepo: ebookRepo, storage: storage}
}

// Upload stores the PDF and records it. The page count is derived from the
// file itself rather than taken from the client, so every screen that shows
// reading progress divides by the same, correct number.
func (s *EbookService) Upload(ctx context.Context, req dto.EbookUploadRequest, file io.ReadSeeker, fileSize int64, fileName string, userID string) (*model.Ebook, error) {
	totalPages, err := pdfutil.PageCount(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPDF, err)
	}

	objectName := fmt.Sprintf("ebooks/%s/%s", userID, fileName)
	key, err := s.storage.UploadFile(ctx, objectName, file, fileSize, "application/pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	isPrivate := true
	if req.IsPrivate != nil {
		isPrivate = *req.IsPrivate
	}

	ebook := &model.Ebook{
		Title:      req.Title,
		Author:     req.Author,
		FileURL:    key,
		FileSize:   fileSize,
		TotalPages: totalPages,
		UploadedBy: userID,
		IsPrivate:  isPrivate,
	}

	if err := s.ebookRepo.Create(ctx, ebook); err != nil {
		return nil, fmt.Errorf("failed to save ebook: %w", err)
	}

	return ebook, nil
}

func canView(e *model.Ebook, userID string, isAdmin bool) bool {
	if isAdmin {
		return true
	}
	if !e.IsPrivate {
		return true
	}
	return e.UploadedBy == userID
}

func canManage(e *model.Ebook, userID string, isAdmin bool) bool {
	return isAdmin || e.UploadedBy == userID
}

func (s *EbookService) GetByID(ctx context.Context, id, userID string, isAdmin bool) (*model.Ebook, error) {
	e, err := s.ebookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canView(e, userID, isAdmin) {
		return nil, ErrForbidden
	}
	return e, nil
}

func (s *EbookService) List(ctx context.Context, page, perPage int, userID string, isAdmin bool) ([]*model.Ebook, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 50 {
		perPage = 20
	}
	return s.ebookRepo.ListVisible(ctx, page, perPage, userID, isAdmin)
}

func (s *EbookService) Update(ctx context.Context, id string, req dto.EbookUpdateRequest, userID string, isAdmin bool) (*model.Ebook, error) {
	ebook, err := s.ebookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canManage(ebook, userID, isAdmin) {
		return nil, ErrForbidden
	}

	ebook.Title = req.Title
	ebook.Author = req.Author
	if req.IsPrivate != nil {
		ebook.IsPrivate = *req.IsPrivate
	}

	if err := s.ebookRepo.Update(ctx, ebook); err != nil {
		return nil, err
	}

	return s.ebookRepo.GetByID(ctx, id)
}

func (s *EbookService) Delete(ctx context.Context, id, userID string, isAdmin bool) error {
	ebook, err := s.ebookRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !canManage(ebook, userID, isAdmin) {
		return ErrForbidden
	}

	if _, err := s.ebookRepo.Delete(ctx, id); err != nil {
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

func (s *EbookService) GetFileURL(ctx context.Context, id, userID string, isAdmin bool) (string, error) {
	ebook, err := s.ebookRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if !canView(ebook, userID, isAdmin) {
		return "", ErrForbidden
	}

	return s.storage.GetPresignedURL(ctx, ebook.FileURL, 1*time.Hour)
}
