package domain

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID              uuid.UUID  `json:"id"`
	GroupID         *uuid.UUID `json:"groupId,omitempty"`
	OwnerUserID     *uuid.UUID `json:"ownerUserId,omitempty"`
	OriginalName    string     `json:"originalName"`
	StorageKey      string     `json:"-"`
	StorageProvider string     `json:"storageProvider"`
	ContentType     string     `json:"contentType"`
	SizeBytes       int64      `json:"sizeBytes"`
	SHA256          string     `json:"sha256"`
	Visibility      string     `json:"visibility"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type ObjectInfo struct {
	Size        int64
	ContentType string
	ETag        string
}

type Backend interface {
	Name() string
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) (ObjectInfo, error)
	Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	URL(key string) string
}

type Upload struct {
	OwnerUserID uuid.UUID
	GroupID     *uuid.UUID
	Filename    string
	ContentType string
	Reader      io.Reader
}

type FileGroup struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Sort      int32     `json:"sort"`
	FileCount int64     `json:"fileCount"`
	CreatedAt time.Time `json:"createdAt"`
}

type Repository interface {
	Create(ctx context.Context, file File) (File, error)
	Get(ctx context.Context, id uuid.UUID) (File, error)
	List(ctx context.Context, ownerID uuid.UUID, allOwners bool, groupID *uuid.UUID, query string, page, pageSize int) ([]File, int64, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	FileGroups(ctx context.Context, ownerID uuid.UUID, allOwners bool) ([]FileGroup, error)
	FileGroup(ctx context.Context, id uuid.UUID) (FileGroup, error)
	CreateFileGroup(ctx context.Context, group FileGroup) (FileGroup, error)
	UpdateFileGroup(ctx context.Context, group FileGroup) (FileGroup, error)
	DeleteFileGroup(ctx context.Context, id uuid.UUID) error
}
