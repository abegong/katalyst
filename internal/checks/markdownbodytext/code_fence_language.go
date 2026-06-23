package markdownbodytext

import (
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// MarkdownCodeFenceHasLanguage checks that fenced code blocks specify a language.
type MarkdownCodeFenceHasLanguage struct{}

func (m MarkdownCodeFenceHasLanguage) Run(ctx checks.Context) []checks.Violation {
	inFence := false
	for _, line := range checks.MarkdownLines(ctx.Doc.Body, ctx.Doc.BodyLine) {
		trimmed := strings.TrimSpace(line.Text)
		if !strings.HasPrefix(trimmed, "```") {
			continue
		}
		if inFence {
			inFence = false
			continue
		}
		lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
		if lang == "" {
			return []checks.Violation{{
				Path:    "/",
				Message: "code fence opening must include a language",
				Line:    line.Line,
			}}
		}
		inFence = true
	}
	return nil
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckMarkdownCodeFenceHasLanguage,
		Family:    "markdownBodyText",
		Slug:      "code-fence-language-required",
		Title:     "Code fence language required",
		Summary:   "Require that opening fenced code blocks include a language tag.",
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_code_fence_language_required`,
	}, func(ch config.CheckInstance) checks.Check {
		return MarkdownCodeFenceHasLanguage{}
	}, nil)
}
