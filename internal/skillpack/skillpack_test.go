package skillpack_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/abegong/katalyst/internal/skillpack"
)

// scaffold writes a minimal skills/ tree into a temp dir: one shippable skill
// with a reference file, one placeholder skill, and the shared bootstrap.
func scaffold(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "katalyst-deploy", "SKILL.md"),
		"---\nname: katalyst-deploy\ndescription: ship me\n---\n\n# Deploy\n")
	writeFile(t, filepath.Join(root, "katalyst-deploy", "references", "pre-commit"),
		"#!/usr/bin/env bash\necho hi\n")

	writeFile(t, filepath.Join(root, "katalyst-migrate-schema", "SKILL.md"),
		"---\nname: katalyst-migrate-schema\nstatus: placeholder\ndescription: nope\n---\n\n# Placeholder\n")

	writeFile(t, filepath.Join(root, "bootstrap.sh"),
		"#!/usr/bin/env bash\necho bootstrap\n")

	return root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func zipEntries(t *testing.T, archive string) map[string]bool {
	t.Helper()
	r, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatalf("open %s: %v", archive, err)
	}
	defer r.Close()
	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	return names
}

func TestPackageAll_shippableOnly_withRootEntrypoint(t *testing.T) {
	skillsDir := scaffold(t)
	outDir := filepath.Join(t.TempDir(), "out")

	artifacts, err := skillpack.PackageAll(skillsDir, outDir)
	if err != nil {
		t.Fatalf("PackageAll: %v", err)
	}

	if len(artifacts) != 1 {
		t.Fatalf("expected 1 shippable artifact, got %d: %v", len(artifacts), artifacts)
	}
	if got := filepath.Base(artifacts[0]); got != "katalyst-deploy.skill" {
		t.Fatalf("artifact name = %q, want katalyst-deploy.skill", got)
	}

	// The placeholder must not be packaged.
	if _, err := os.Stat(filepath.Join(outDir, "katalyst-migrate-schema.skill")); !os.IsNotExist(err) {
		t.Fatalf("placeholder skill was packaged; want it skipped")
	}

	entries := zipEntries(t, artifacts[0])
	for _, want := range []string{"SKILL.md", "references/pre-commit", skillpack.BootstrapName} {
		if !entries[want] {
			t.Errorf("archive missing %q at expected path; entries: %v", want, keys(entries))
		}
	}
	// SKILL.md must be at the root, not nested under katalyst-deploy/.
	if entries["katalyst-deploy/SKILL.md"] {
		t.Errorf("SKILL.md is nested under the skill name; want it at the archive root")
	}
}

func TestPackage_placeholder_errors(t *testing.T) {
	skillsDir := scaffold(t)
	outDir := t.TempDir()
	if _, err := skillpack.Package(skillsDir, "katalyst-migrate-schema", outDir); err == nil {
		t.Fatal("Package on a placeholder: want error, got nil")
	}
}

func TestPackage_unknown_errors(t *testing.T) {
	skillsDir := scaffold(t)
	outDir := t.TempDir()
	if _, err := skillpack.Package(skillsDir, "katalyst-nope", outDir); err == nil {
		t.Fatal("Package on an unknown skill: want error, got nil")
	}
}

func TestPackage_singleShippable(t *testing.T) {
	skillsDir := scaffold(t)
	outDir := t.TempDir()
	artifact, err := skillpack.Package(skillsDir, "katalyst-deploy", outDir)
	if err != nil {
		t.Fatalf("Package: %v", err)
	}
	if got := filepath.Base(artifact); got != "katalyst-deploy.skill" {
		t.Fatalf("artifact = %q, want katalyst-deploy.skill", got)
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
