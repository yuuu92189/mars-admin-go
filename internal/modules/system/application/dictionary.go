package application

import (
	"context"
	"strings"

	"mars-admin-go/internal/modules/system/domain"
	"mars-admin-go/internal/platform/apperror"

	"github.com/google/uuid"
)

type DictionaryService struct {
	repo  domain.Repository
	cache domain.Cache
}

func NewDictionaryService(repo domain.Repository, caches ...domain.Cache) *DictionaryService {
	service := &DictionaryService{repo: repo}
	if len(caches) > 0 {
		service.cache = caches[0]
	}
	return service
}

func (s *DictionaryService) Types(ctx context.Context) ([]domain.DictionaryType, error) {
	var cached []domain.DictionaryType
	if s.cache != nil {
		if hit, _ := s.cache.Get(ctx, "dictionary:types", &cached); hit {
			return cached, nil
		}
	}
	items, err := s.repo.DictionaryTypes(ctx)
	if err == nil && s.cache != nil {
		_ = s.cache.Set(ctx, "dictionary:types", items)
	}
	return items, err
}

func (s *DictionaryService) Items(ctx context.Context, typeID uuid.UUID) ([]domain.DictionaryItem, error) {
	key := "dictionary:items:" + typeID.String()
	var cached []domain.DictionaryItem
	if s.cache != nil {
		if hit, _ := s.cache.Get(ctx, key, &cached); hit {
			return cached, nil
		}
	}
	items, err := s.repo.DictionaryItems(ctx, typeID)
	if err == nil && s.cache != nil {
		_ = s.cache.Set(ctx, key, items)
	}
	return items, err
}

func (s *DictionaryService) CreateType(ctx context.Context, input domain.DictionaryTypeInput) (domain.DictionaryType, error) {
	if err := validateDictionaryType(&input); err != nil {
		return domain.DictionaryType{}, err
	}
	item, err := s.repo.CreateDictionaryType(ctx, input)
	s.invalidate(ctx)
	return item, err
}

func (s *DictionaryService) UpdateType(ctx context.Context, id uuid.UUID, input domain.DictionaryTypeInput) (domain.DictionaryType, error) {
	if err := validateDictionaryType(&input); err != nil {
		return domain.DictionaryType{}, err
	}
	item, err := s.repo.UpdateDictionaryType(ctx, id, input)
	s.invalidate(ctx)
	return item, err
}

func (s *DictionaryService) DeleteType(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteDictionaryType(ctx, id)
	s.invalidate(ctx)
	return err
}

func (s *DictionaryService) CreateItem(ctx context.Context, input domain.DictionaryItemInput) (domain.DictionaryItem, error) {
	if err := validateDictionaryItem(&input); err != nil {
		return domain.DictionaryItem{}, err
	}
	item, err := s.repo.CreateDictionaryItem(ctx, input)
	s.invalidate(ctx)
	return item, err
}

func (s *DictionaryService) UpdateItem(ctx context.Context, id uuid.UUID, input domain.DictionaryItemInput) (domain.DictionaryItem, error) {
	if err := validateDictionaryItem(&input); err != nil {
		return domain.DictionaryItem{}, err
	}
	item, err := s.repo.UpdateDictionaryItem(ctx, id, input)
	s.invalidate(ctx)
	return item, err
}

func (s *DictionaryService) DeleteItem(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteDictionaryItem(ctx, id)
	s.invalidate(ctx)
	return err
}

func (s *DictionaryService) invalidate(ctx context.Context) {
	if s.cache != nil {
		_ = s.cache.DeletePrefix(ctx, "dictionary:")
	}
}

func validateDictionaryType(input *domain.DictionaryTypeInput) error {
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	if input.Code == "" || len(input.Code) > 100 {
		return apperror.Validation("code", "Dictionary type code is required and must not exceed 100 characters.")
	}
	if input.Name == "" {
		return apperror.Validation("name", "Dictionary type name is required.")
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Status != "active" && input.Status != "disabled" {
		return apperror.Validation("status", "status must be active or disabled.")
	}
	return nil
}

func validateDictionaryItem(input *domain.DictionaryItemInput) error {
	input.Value = strings.TrimSpace(input.Value)
	input.Label = strings.TrimSpace(input.Label)
	if input.Value == "" || len(input.Value) > 200 {
		return apperror.Validation("value", "Dictionary item value is required and must not exceed 200 characters.")
	}
	if input.Label == "" || len(input.Label) > 200 {
		return apperror.Validation("label", "Dictionary item label is required and must not exceed 200 characters.")
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Status != "active" && input.Status != "disabled" {
		return apperror.Validation("status", "status must be active or disabled.")
	}
	return nil
}
