package examples_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/abegong/katalyst/internal/examples"
)

// updateGoldens rewrites the per-example fixtures instead of asserting:
//
//	go test ./internal/examples -run TestExamples -update
//
// the canonical Go golden-file pattern. Generate, then review the diff. These
// goldens are what makes each example's output a tested contract: cmd/gendocs
// renders the same Run output into the docs, so a behavior change that alters an
// example fails here loudly.
var updateGoldens = flag.Bool("update", false, "rewrite example goldens")

func TestExamples(t *testing.T) {
	for _, ex := range examples.All() {
		ex := ex
		t.Run(ex.ID, func(t *testing.T) {
			res, err := examples.Run(ex)
			if err != nil {
				t.Fatalf("run %s: %v", ex.ID, err)
			}
			got := examples.RenderPage(ex, res)
			golden := filepath.Join("testdata", ex.ID+".md")
			if *updateGoldens {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("missing golden for %s (run with -update): %v", ex.ID, err)
			}
			if got != string(want) {
				t.Errorf("example %s output drifted (run with -update to accept).\n--- got ---\n%s\n--- want ---\n%s", ex.ID, got, want)
			}
		})
	}
}

// TestExamples_uniqueIDs guards the registry: IDs are slugs used as file paths
// and shortcode arguments, so collisions would silently overwrite output.
func TestExamples_uniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, ex := range examples.All() {
		if ex.ID == "" {
			t.Errorf("example with empty ID: %q", ex.Title)
		}
		if seen[ex.ID] {
			t.Errorf("duplicate example ID %q", ex.ID)
		}
		seen[ex.ID] = true
	}
}
