package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/Anjsvf/read-img-go/domain"
	"github.com/Anjsvf/read-img-go/repository"
	"github.com/google/uuid"
)

type MeasureService interface {
	Upload(ctx context.Context, req *domain.UploadRequest) (*domain.UploadResponse, error)
	Confirm(ctx context.Context, req *domain.ConfirmRequest) error
	List(ctx context.Context, customerCode string, measureType *domain.MeasureType) (*domain.ListResponse, error)
}

// Sentinel errors used by handlers to map to correct HTTP status codes.
var (
	ErrDoubleReport          = fmt.Errorf("DOUBLE_REPORT")
	ErrMeasureNotFound       = fmt.Errorf("MEASURE_NOT_FOUND")
	ErrConfirmationDuplicate = fmt.Errorf("CONFIRMATION_DUPLICATE")
	ErrMeasuresNotFound      = fmt.Errorf("MEASURES_NOT_FOUND")
	ErrInvalidBase64         = fmt.Errorf("INVALID_BASE64")
)

type measureSvc struct {
	repo       repository.MeasureRepository
	gemini     GeminiService
	cloudinary CloudinaryService
}

func NewMeasureService(repo repository.MeasureRepository, gemini GeminiService, cloudinary CloudinaryService) MeasureService {
	return &measureSvc{
		repo:       repo,
		gemini:     gemini,
		cloudinary: cloudinary,
	}
}

func (s *measureSvc) Upload(ctx context.Context, req *domain.UploadRequest) (*domain.UploadResponse, error) {
	// Validate base64 (strip data URI prefix first if present)
	rawBase64 := req.Image
	if idx := strings.Index(rawBase64, ";base64,"); idx != -1 {
		rawBase64 = rawBase64[idx+8:]
	}
	if _, err := base64.StdEncoding.DecodeString(rawBase64); err != nil {
		// Try URL-safe encoding
		if _, err2 := base64.URLEncoding.DecodeString(rawBase64); err2 != nil {
			return nil, ErrInvalidBase64
		}
	}

	// Validate measure type
	mt := domain.MeasureType(strings.ToUpper(string(req.MeasureType)))
	if !mt.IsValid() {
		return nil, fmt.Errorf("invalid measure_type: must be WATER or GAS")
	}

	// Check for double report in the same month
	exists, err := s.repo.ExistsByTypeAndMonth(ctx, req.CustomerCode, mt, req.MeasureDatetime)
	if err != nil {
		return nil, fmt.Errorf("check existing measure: %w", err)
	}
	if exists {
		return nil, ErrDoubleReport
	}

	// Extract meter value via Gemini (base64 stays in memory only)
	value, err := s.gemini.ExtractMeterValue(ctx, req.Image)
	if err != nil {
		return nil, fmt.Errorf("gemini extraction: %w", err)
	}

	// Upload image to Cloudinary, get public URL (base64 is never persisted)
	imageURL, err := s.cloudinary.Upload(ctx, req.Image)
	if err != nil {
		return nil, fmt.Errorf("cloudinary upload: %w", err)
	}

	// Build and persist measure (no base64 stored)
	measure := &domain.Measure{
		MeasureUUID:     uuid.NewString(),
		CustomerCode:    req.CustomerCode,
		MeasureType:     mt,
		MeasureDatetime: req.MeasureDatetime.UTC(),
		ImageURL:        imageURL, // Cloudinary public URL only
		MeasureValue:    value,
		HasConfirmed:    false,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, measure); err != nil {
		return nil, fmt.Errorf("save measure: %w", err)
	}

	return &domain.UploadResponse{
		ImageURL:     measure.ImageURL,
		MeasureValue: measure.MeasureValue,
		MeasureUUID:  measure.MeasureUUID,
	}, nil
}

func (s *measureSvc) Confirm(ctx context.Context, req *domain.ConfirmRequest) error {
	measure, err := s.repo.FindByUUID(ctx, req.MeasureUUID)
	if err != nil {
		return fmt.Errorf("find measure: %w", err)
	}
	if measure == nil {
		return ErrMeasureNotFound
	}
	if measure.HasConfirmed {
		return ErrConfirmationDuplicate
	}

	if err := s.repo.Confirm(ctx, req.MeasureUUID, req.ConfirmedValue); err != nil {
		return fmt.Errorf("confirm measure: %w", err)
	}
	return nil
}

func (s *measureSvc) List(ctx context.Context, customerCode string, measureType *domain.MeasureType) (*domain.ListResponse, error) {
	measures, err := s.repo.ListByCustomer(ctx, customerCode, measureType)
	if err != nil {
		return nil, fmt.Errorf("list measures: %w", err)
	}
	if len(measures) == 0 {
		return nil, ErrMeasuresNotFound
	}

	items := make([]domain.MeasureListItem, 0, len(measures))
	for _, m := range measures {
		items = append(items, domain.MeasureListItem{
			MeasureUUID:     m.MeasureUUID,
			MeasureDatetime: m.MeasureDatetime,
			MeasureType:     m.MeasureType,
			HasConfirmed:    m.HasConfirmed,
			ImageURL:        m.ImageURL,
		})
	}

	return &domain.ListResponse{
		CustomerCode: customerCode,
		Measures:     items,
	}, nil
}
