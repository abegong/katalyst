package cmd_test

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// The snapshot harness backs every full-output fixture comparison in the cmd
// suite. Text contracts (help, list/show output, the inspect report, canonical
// stderr diagnostics) are pinned as golden files under testdata/snapshots/ and
// reviewed as plain text; behavior stays in property tests. See
// cmd/AGENTS.md ("Testing the CLI").

// updateSnapshots rewrites fixtures instead of asserting:
//
//	go test ./cmd -run TestThing -update
//
// the canonical Go golden-file pattern. Generate, then review the diff.
var updateSnapshots = flag.Bool("update", false, "rewrite snapshot fixtures")

// pkgDir is the directory holding this test file, used to locate the source
// testdata/ tree for -update writes. os.Getwd is unreliable mid-test because
// the CLI helpers chdir into t.TempDir(); reads go through the embedded FS
// (snapshotFixtures in fixtures_test.go) for the same reason.
func pkgDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("snapshot: runtime.Caller failed")
	}
	return filepath.Dir(file)
}

// snapshot asserts got equals the fixture at testdata/snapshots/<name> (a slash
// path, e.g. "collection/list.txt"), after applying each norm to got. With
// -update it rewrites the fixture (in normalized form) instead of asserting.
func snapshot(t *testing.T, name, got string, norm ...func(string) string) {
	t.Helper()
	got = applyNorm(got, norm...)
	if *updateSnapshots {
		if err := writeSnapshot(filepath.Join(pkgDir(), "testdata/snapshots"), name, got); err != nil {
			t.Fatalf("snapshot %q: write: %v", name, err)
		}
		return
	}
	ok, want, err := matchSnapshot(name, got)
	if err != nil {
		t.Fatalf("snapshot %q: missing fixture (re-run with -update): %v", name, err)
	}
	if !ok {
		t.Errorf("snapshot %q mismatch (re-run with -update to accept).\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

// applyNorm folds the normalizers over s left to right.
func applyNorm(s string, norm ...func(string) string) string {
	for _, fn := range norm {
		s = fn(s)
	}
	return s
}

// matchSnapshot reads the embedded fixture and reports whether got equals it.
func matchSnapshot(name, got string) (ok bool, want string, err error) {
	b, err := snapshotFixtures.ReadFile("testdata/snapshots/" + name)
	if err != nil {
		return false, "", err
	}
	return got == string(b), string(b), nil
}

// writeSnapshot writes got to base/<name>, creating parent directories. The
// -update path and the harness self-tests share it (the latter with a temp
// base).
func writeSnapshot(base, name, got string) error {
	path := filepath.Join(base, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(got), 0o644)
}

// normTmp rewrites every occurrence of a test's temp dir to a stable token, so
// output that embeds an absolute path (check diagnostics, the inspect report
// header) snapshots deterministically.
func normTmp(dir string) func(string) string {
	aliases := []string{dir}
	if realDir, err := filepath.EvalSymlinks(dir); err == nil {
		aliases = append(aliases, realDir)
	}
	for _, alias := range aliases[:] {
		switch {
		case strings.HasPrefix(alias, "/var/"):
			aliases = append(aliases, "/private"+alias)
		case strings.HasPrefix(alias, "/private/var/"):
			aliases = append(aliases, strings.TrimPrefix(alias, "/private"))
		}
	}
	sort.Slice(aliases, func(i, j int) bool { return len(aliases[i]) > len(aliases[j]) })
	return func(s string) string {
		for _, alias := range aliases {
			s = strings.ReplaceAll(s, alias, "<project>")
		}
		return s
	}
}

// --- harness self-coverage ---

func TestSnapshot_matchesFixture(t *testing.T) {
	ok, want, err := matchSnapshot("selftest/match.txt", "hello\n")
	if err != nil {
		t.Fatalf("matchSnapshot: %v", err)
	}
	if !ok {
		t.Errorf("expected match against selftest fixture, want=%q", want)
	}
}

func TestSnapshot_detectsMismatch(t *testing.T) {
	ok, _, err := matchSnapshot("selftest/match.txt", "different\n")
	if err != nil {
		t.Fatalf("matchSnapshot: %v", err)
	}
	if ok {
		t.Errorf("expected mismatch to be reported")
	}
}

func TestSnapshot_missingFixtureIsError(t *testing.T) {
	if _, _, err := matchSnapshot("selftest/does-not-exist.txt", "x"); err == nil {
		t.Errorf("expected an error for a missing fixture")
	}
}

func TestSnapshot_normTmpRewritesPath(t *testing.T) {
	got := normTmp("/tmp/abc123")("/tmp/abc123/notes/bad.md:3: /year: error\n")
	if want := "<project>/notes/bad.md:3: /year: error\n"; got != want {
		t.Errorf("normTmp = %q, want %q", got, want)
	}
}

func TestSnapshot_normTmpRewritesMacOSTmpSymlinkPath(t *testing.T) {
	got := normTmp("/var/folders/abc123")("/private/var/folders/abc123/notes/bad.md:3: /year: error\n")
	if want := "<project>/notes/bad.md:3: /year: error\n"; got != want {
		t.Errorf("normTmp = %q, want %q", got, want)
	}
}

func TestSnapshot_updateRoundTrip(t *testing.T) {
	base := t.TempDir()
	body := applyNorm("/tmp/xyz/notes/x.md: OK\n", normTmp("/tmp/xyz"))
	if err := writeSnapshot(base, "round/trip.txt", body); err != nil {
		t.Fatalf("writeSnapshot: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(base, "round", "trip.txt"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(b) != "<project>/notes/x.md: OK\n" {
		t.Errorf("round-trip stored %q, want normalized form", b)
	}
}
