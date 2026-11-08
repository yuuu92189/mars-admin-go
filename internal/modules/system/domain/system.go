package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DictionaryType struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	ItemCount   int64     `json:"itemCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

type DictionaryItem struct {
	ID        uuid.UUID `json:"id"`
	TypeID    uuid.UUID `json:"typeId"`
	Value     string    `json:"value"`
	Label     string    `json:"label"`
	Sort      int32     `json:"sort"`
	TagStyle  string    `json:"tagStyle"`
	IsDefault bool      `json:"isDefault"`
	Status    string    `json:"status"`
}

type DictionaryTypeInput struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type DictionaryItemInput struct {
	TypeID    uuid.UUID `json:"typeId"`
	Value     string    `json:"value"`
	Label     string    `json:"label"`
	Sort      int32     `json:"sort"`
	TagStyle  string    `json:"tagStyle"`
	IsDefault bool      `json:"isDefault"`
	Status    string    `json:"status"`
}

type SettingGroup struct {
	ID           uuid.UUID      `json:"id"`
	Code         string         `json:"code"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Sort         int32          `json:"sort"`
	Status       string         `json:"status"`
	SettingCount int64          `json:"settingCount,omitempty"`
	Settings     []Setting      `json:"settings,omitempty"`
	Values       map[string]any `json:"values,omitempty"`
}

type Setting struct {
	ID        uuid.UUID `json:"id"`
	Key       string    `json:"key"`
	ValueType string    `json:"valueType"`
	Value     any       `json:"value"`
	Sensitive bool      `json:"sensitive"`
	Version   int64     `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
	RawValue  []byte    `json:"-"`
}

type SettingUpdate struct {
	Key       string
	ValueType string
	Raw       []byte
	Sensitive bool
}

type Repository interface {
	DictionaryTypes(ctx context.Context) ([]DictionaryType, error)
	DictionaryItems(ctx context.Context, typeID uuid.UUID) ([]DictionaryItem, error)
	CreateDictionaryType(ctx context.Context, input DictionaryTypeInput) (DictionaryType, error)
	UpdateDictionaryType(ctx context.Context, id uuid.UUID, input DictionaryTypeInput) (DictionaryType, error)
	DeleteDictionaryType(ctx context.Context, id uuid.UUID) error
	CreateDictionaryItem(ctx context.Context, input DictionaryItemInput) (DictionaryItem, error)
	UpdateDictionaryItem(ctx context.Context, id uuid.UUID, input DictionaryItemInput) (DictionaryItem, error)
	DeleteDictionaryItem(ctx context.Context, id uuid.UUID) error
	SettingGroups(ctx context.Context) ([]SettingGroup, error)
	SettingGroupByCode(ctx context.Context, code string, includeSensitive bool) (SettingGroup, error)
	PublicSettings(ctx context.Context) (map[string]any, error)
	UpsertSettings(ctx context.Context, groupCode string, updates []SettingUpdate, changedBy uuid.UUID) error
}

type Cache interface {
	Get(ctx context.Context, key string, destination any) (bool, error)
	Set(ctx context.Context, key string, value any) error
	DeletePrefix(ctx context.Context, suffix string) error
}
