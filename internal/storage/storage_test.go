package storage_test

import (
	"testing"

	"github.com/abegong/katalyst/internal/storage"
)

func TestKnown_implementedBackends(t *testing.T) {
	if !storage.Known(storage.Filesystem) {
		t.Errorf("filesystem should be a known storage type")
	}
	if !storage.Known(storage.SQLite) {
		t.Errorf("sqlite should be a known storage type")
	}
}
