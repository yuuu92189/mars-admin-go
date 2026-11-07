package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mars-admin-go/internal/modules/storage/domain"
	"mars-admin-go/internal/platform/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo      domain.Repository
	backend   domain.Backend
	maxUpload int64
	now       func() time.Time
}

func New(repo domain.Repository, backend domain.Backend, maxUpload int64) *Service {
	if maxUpload <= 0 {
		maxUpload = 50 << 20
	}
	return &Service{repo: repo, backend: backend, maxUpload: maxUpload, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Upload(ctx context.Context, input domain.Upload) (domain.File, error) {
	name := safeDisplayName(input.Filename)
	if name == "" {
		return domain.File{}, apperror.Validation("file", "A valid file name is required.")
	}
	tmp, err := os.CreateTemp("", "mars-upload-*")
	if err != nil {
		return domain.File{}, fmt.Errorf("create upload staging file: %w", err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), io.LimitReader(input.Reader, s.maxUpload+1))
	if err != nil {
		return domain.File{}, fmt.Errorf("stage upload: %w", err)
	}
	if written > s.maxUpload {
		return domain.File{}, apperror.New(apperror.KindValidation, "file_too_large", fmt.Sprintf("File exceeds the %d byte limit.", s.maxUpload))
	}
	if written == 0 {
		return domain.File{}, apperror.Validation("file", "Empty files are not allowed.")
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return domain.File{}, err
	}
	header := make([]byte, 512)
	n, _ := io.ReadFull(tmp, header)
	header = header[:n]
	detected := http.DetectContentType(header)
	if input.ContentType != "" && input.ContentType != "application/octet-stream" && !compatibleTypes(input.ContentType, detected) {
		return domain.File{}, apperror.Validation("file", "Declared content type does not match file content.")
	}
	contentType := detected
	if err := validateContentType(name, contentType); err != nil {
		return domain.File{}, err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return domain.File{}, err
	}

	now := s.now()
	key := filepath.ToSlash(filepath.Join(now.Format("2006"), now.Format("01"), uuid.Must(uuid.NewV7()).String()))
	info, err := s.backend.Put(ctx, key, tmp, written, contentType)
	if err != nil {
		return domain.File{}, fmt.Errorf("store file: %w", err)
	}
	file := domain.File{
		ID:              mustID(),
		GroupID:         input.GroupID,
		OwnerUserID:     &input.OwnerUserID,
		OriginalName:    name,
		StorageKey:      key,
		StorageProvider: s.backend.Name(),
		ContentType:     contentType,
		SizeBytes:       info.Size,
		SHA256:          hex.EncodeToString(hasher.Sum(nil)),
		Visibility:      "private",
		Status:          "active",
		CreatedAt:       now,
	}
	if file.SizeBytes <= 0 {
		file.SizeBytes = written
	}
	created, err := s.repo.Create(ctx, file)
	if err != nil {
		_ = s.backend.Delete(ctx, key)
		return domain.File{}, err
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.File, error) {
	file, err := s.repo.Get(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.File{}, apperror.NotFound("File")
	}
	return file, err
}

func (s *Service) Open(ctx context.Context, id uuid.UUID) (domain.File, io.ReadCloser, domain.ObjectInfo, error) {
	file, err := s.Get(ctx, id)
	if err != nil {
		return domain.File{}, nil, domain.ObjectInfo{}, err
	}
	reader, info, err := s.backend.Get(ctx, file.StorageKey)
	if err != nil {
		return domain.File{}, nil, domain.ObjectInfo{}, fmt.Errorf("read stored file: %w", err)
	}
	return file, reader, info, nil
}

func (s *Service) List(ctx context.Context, groupID *uuid.UUID, query string, page, pageSize int) ([]domain.File, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, groupID, strings.TrimSpace(query), page, pageSize)
}

func (s *Service) FileGroups(ctx context.Context) ([]domain.FileGroup, error) {
	return s.repo.FileGroups(ctx)
}

func (s *Service) FileGroup(ctx context.Context, id uuid.UUID) (domain.FileGroup, error) {
	return s.repo.FileGroup(ctx, id)
}

func (s *Service) CreateFileGroup(ctx context.Context, name string, sort int32) (domain.FileGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return domain.FileGroup{}, apperror.Validation("name", "File group name must contain 1 to 100 characters.")
	}
	return s.repo.CreateFileGroup(ctx, domain.FileGroup{ID: uuid.New(), Name: name, Sort: sort})
}

func (s *Service) UpdateFileGroup(ctx context.Context, id uuid.UUID, name string, sort int32) (domain.FileGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return domain.FileGroup{}, apperror.Validation("name", "File group name must contain 1 to 100 characters.")
	}
	return s.repo.UpdateFileGroup(ctx, domain.FileGroup{ID: id, Name: name, Sort: sort})
}

func (s *Service) DeleteFileGroup(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteFileGroup(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	file, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	if err := s.backend.Delete(ctx, file.StorageKey); err != nil {
		return fmt.Errorf("delete file object: %w", err)
	}
	return nil
}

func (s *Service) URL(file domain.File) string {
	return fmt.Sprintf("/api/v1/files/%s/download", file.ID)
}

func safeDisplayName(name string) string {
	name = filepath.Base(strings.TrimSpace(strings.ReplaceAll(name, "\\", "/")))
	name = strings.ReplaceAll(name, "\x00", "")
	if name == "." || name == ".." || name == "" {
		return ""
	}
	runes := []rune(name)
	if len(runes) > 200 {
		name = string(runes[:200])
	}
	return name
}

func compatibleTypes(declared, detected string) bool {
	declaredType, _, _ := mime.ParseMediaType(declared)
	detectedType, _, _ := mime.ParseMediaType(detected)
	if declaredType == detectedType {
		return true
	}
	if strings.HasPrefix(declaredType, "text/") && strings.HasPrefix(detectedType, "text/") {
		return true
	}
	if declaredType == "application/octet-stream" {
		return true
	}
	if declaredType == "application/zip" {
		return detectedType == "application/zip" || detectedType == "application/octet-stream"
	}
	if declaredType == "application/pdf" && detectedType == "application/pdf" {
		return true
	}
	return false
}

func validateContentType(name, contentType string) error {
	ext := strings.ToLower(filepath.Ext(name))
	allowedExtensions := map[string]struct{}{
		".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".webp": {},
		".pdf": {}, ".txt": {}, ".md": {}, ".csv": {}, ".json": {}, ".yaml": {}, ".yml": {},
		".doc": {}, ".docx": {}, ".xls": {}, ".xlsx": {}, ".ppt": {}, ".pptx": {},
		".mp4": {}, ".webm": {}, ".mov": {}, ".mp3": {}, ".wav": {}, ".ogg": {},
		".zip": {}, ".7z": {}, ".tar": {}, ".gz": {},
	}
	if _, ok := allowedExtensions[ext]; !ok {
		return apperror.Validation("file", "This file extension is not allowed.")
	}
	if contentType == "text/html" || contentType == "image/svg+xml" ||
		strings.Contains(contentType, "javascript") || strings.Contains(contentType, "xml") {
		return apperror.Validation("file", "Executable or active web content is not allowed.")
	}
	return nil
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
