package inspect_test

import (
	"reflect"
	"testing"

	"github.com/abegong/katalyst/internal/inspect"
)

func TestParseSelection(t *testing.T) {
	tests := []struct {
		raw  string
		want inspect.Selection
	}{
		{"", inspect.Selection{Label: "all files", Mode: inspect.SelectionAll}},
		{"content/books/*.md", inspect.Selection{Label: "content/books/*.md", Mode: inspect.SelectionGlob, Pattern: "content/books/*.md"}},
		{`ext = ".csv"`, inspect.Selection{Label: `ext = ".csv"`, Mode: inspect.SelectionExt, Pattern: ".csv"}},
		{`path under "docs/reference"`, inspect.Selection{Label: `path under "docs/reference"`, Mode: inspect.SelectionPathUnder, Pattern: "docs/reference"}},
		{"raw/notes", inspect.Selection{Label: "raw/notes", Mode: inspect.SelectionDir, Pattern: "raw/notes"}},
	}
	for _, tt := range tests {
		if got := inspect.ParseSelection(tt.raw); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ParseSelection(%q) = %#v, want %#v", tt.raw, got, tt.want)
		}
	}
}
