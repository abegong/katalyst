package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
)

// fakeUnavailableLib backs a fake non-object check type with an out-of-process
// library that is never available, so the engine's availability gate can be
// exercised without a real external tool.
type fakeUnavailableLib struct{}

func (fakeUnavailableLib) Name() string     { return "fake-prose" }
func (fakeUnavailableLib) Available() error { return errors.New("binary not found") }

func init() {
	// Registered from init (not a test body) so it never races test ordering
	// or leaks into a test that has already snapshotted the registry.
	checks.RegisterLibrary(fakeUnavailableLib{})
	checks.RegisterDescriptor(checks.Descriptor{
		CheckType:     "fake_prose_unavailable",
		Library:       "fake-prose",
		Family:        "plainText",
		Slug:          "fake-prose-unavailable",
		Title:         "Fake Prose",
		Summary:       "Test-only out-of-process check type.",
		ConfigExample: "checks:\n  - kind: fake_prose_unavailable",
	})
}

// A non-object check whose library is unavailable fails the run with a clear,
// library-named error before any item is checked.
func TestEnsureLibrariesAvailable_unavailableFails(t *testing.T) {
	err := ensureLibrariesAvailable([]checks.ConfiguredCheck{{Kind: "fake_prose_unavailable"}})
	if err == nil {
		t.Fatal("expected an unavailable-library error")
	}
	if !strings.Contains(err.Error(), "fake-prose") || !strings.Contains(err.Error(), "unavailable") {
		t.Errorf("error should name the library and its state, got: %v", err)
	}
}

// Native check types name no library, and json-schema is always available, so
// the gate never misfires on the common case.
func TestEnsureLibrariesAvailable_nativeAndObjectPass(t *testing.T) {
	err := ensureLibrariesAvailable([]checks.ConfiguredCheck{
		{Kind: checks.CheckMarkdownSingleH1},
		{Kind: checks.CheckObject},
	})
	if err != nil {
		t.Errorf("native and object kinds should pass availability, got %v", err)
	}
}
