package storage_test

import (
	"testing"

	"github.com/abegong/katalyst/internal/storage"
)

func TestKnown_onlyFilesystem(t *testing.T) {
	if !storage.Known(storage.Filesystem) {
		t.Errorf("filesystem should be a known storage type")
	}
	if storage.Known(storage.BaseType("sqlite")) {
		t.Errorf("sqlite is not implemented yet and should not be known")
	}
}
