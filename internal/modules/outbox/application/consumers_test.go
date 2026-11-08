package application

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"mars-admin-go/internal/modules/outbox/contracts"
	storagedomain "mars-admin-go/internal/modules/storage/domain"
	platformoutbox "mars-admin-go/internal/platform/outbox"

	"github.com/google/uuid"
)

type failingStorageBackend struct{}

func (failingStorageBackend) Name() string { return "local" }
func (failingStorageBackend) Put(context.Context, string, io.Reader, int64, string) (storagedomain.ObjectInfo, error) {
	return storagedomain.ObjectInfo{}, errors.New("not used")
}
func (failingStorageBackend) Get(context.Context, string) (io.ReadCloser, storagedomain.ObjectInfo, error) {
	return nil, storagedomain.ObjectInfo{}, errors.New("not used")
}
func (failingStorageBackend) Delete(context.Context, string) error {
	return errors.New("simulated object deletion failure")
}
func (failingStorageBackend) Exists(context.Context, string) (bool, error) {
	return false, errors.New("not used")
}
func (failingStorageBackend) URL(string) string { return "" }

func TestFileDeletionConsumerReturnsBackendFailureForRetry(t *testing.T) {
	fileID := uuid.New()
	payload, err := json.Marshal(contracts.FileDeleted{
		FileID: fileID, StorageKey: "2026/11/file", StorageProvider: "local",
	})
	if err != nil {
		t.Fatal(err)
	}
	consumers := NewConsumers(nil, nil, failingStorageBackend{})
	err = consumers.HandleFileDeleted(context.Background(), platformoutbox.Event{
		ID: uuid.New(), EventType: contracts.EventFileDeleted, Payload: payload,
	})
	if err == nil {
		t.Fatal("expected object deletion failure to be returned for River retry")
	}
}
