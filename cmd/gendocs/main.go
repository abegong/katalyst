// Command gendocs renders the check-type reference under
// docs/reference/check-types/ from the check descriptors in
// internal/checks/registry.go, and the inspector reference under
// docs/reference/inspectors/ from internal/inspect/registry.go. From
// internal/examples it runs each worked example's command and writes two
// embeddable snippets under docs/generated/examples/ (<id>.txt for output and
// <id>.full.md for the full corpus+command+output), and it embeds the
// feature-demonstrating examples into the generated reference page that owns
// each feature; the rest are embedded by hand into how-to and deep-dive pages.
// It also mirrors the
// repo-root governance files (CODE_OF_CONDUCT.md, SECURITY.md) into
// docs/content/contributing/ so the published site carries them without a
// second source of truth: the root files stay canonical (GitHub surfaces them
// from the repo root), the docs copies are generated. Run via `make docs-gen`.
// CI fails if the working tree drifts from its output, so the registries, the
// example registry, and the root files are the single source of truth for that
// documentation.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	_ "github.com/abegong/katalyst/internal/checks/all" // register every check-type family
	"github.com/abegong/katalyst/internal/examples"
	"github.com/abegong/katalyst/internal/inspect"
)

// outDir is the generated check-types section, relative to the repo root.
const outDir = "docs/content/reference/check-types"

// inspectorsOut is the generated inspectors section, relative to the repo root.
const inspectorsOut = "docs/content/reference/inspectors"

// examplesOut is the retired worked-examples catalog section; gendocs removes
// it so the directory does not linger after examples moved inline.
// examplesSnippetsOut holds the embeddable snippets the {{< katalyst-example >}}
// and {{< katalyst-example-full >}} shortcodes pull into prose pages; it lives
// outside content/ so Hugo does not render it and katalyst does not check it.
const (
	examplesOut         = "docs/content/reference/examples"
	examplesSnippetsOut = "docs/generated/examples"
)

// examplesByPage maps a generated reference page to the worked example embedded
// into it as a "## Worked example" section. The key identifies the page:
// "checktype:<family>/<slug>", "family:<family>", or "inspector:<layer>/<slug>".
// Examples not listed here are embedded by hand into how-to and deep-dive pages
// via the {{< katalyst-example-full >}} shortcode (the command/workflow bucket).
var examplesByPage = map[string]string{
	"checktype:structured-object/field-type":        "check-type-error",
	"checktype:structured-object/required-field":    "check-schema-missing-field",
	"family:structured-object":                      "check-valid-item",
	"checktype:markdown-body-text/title-matches-h1": "check-title-h1-mismatch",
	"checktype:plain-text/forbids":                  "fix-text-forbids",
	"inspector:source/document-shape":               "inspect-source-shape",
	"inspector:collection/object-fields":            "inspect-collection-fields",
}

// workedExample returns the "## Worked example" block for a generated page, or
// "" if no example is homed there. The block defers to the embeddable snippet so
// the rendered example lives in exactly one generated artifact.
func workedExample(pageKey string) string {
	id, ok := examplesByPage[pageKey]
	if !ok {
		return ""
	}
	return fmt.Sprintf("\n## Worked example\n\n{{< katalyst-example-full \"%s\" >}}\n", id)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gendocs:", err)
		os.Exit(1)
	}
}

func run() error {
	// Remove the existing generated tree so deletions propagate, then
	// rewrite it from scratch.
	if err := os.RemoveAll(outDir); err != nil {
		return err
	}

	families := checks.Families()
	byFamily := map[string][]checks.Descriptor{}
	for _, d := range checks.Descriptors() {
		byFamily[d.Family] = append(byFamily[d.Family], d)
	}

	// Section landing page.
	if err := write(filepath.Join(outDir, "_index.md"), sectionIndex(families, byFamily)); err != nil {
		return err
	}

	for fi, fam := range families {
		dir := filepath.Join(outDir, fam.Slug)
		ds := byFamily[fam.ID]
		// Family landing page; weight orders families within the section.
		if err := write(filepath.Join(dir, "_index.md"), familyIndex(fam, ds, (fi+1)*10)); err != nil {
			return err
		}
		for di, d := range ds {
			if err := write(filepath.Join(dir, d.Slug+".md"), checkTypePage(d, fam, (di+1)*10)); err != nil {
				return err
			}
		}
	}

	// Inspectors reference: a section index, a landing page per layer, and one
	// page per inspector, mirroring the check-types tree so `inspectors show`
	// detail is documented and discoverable, not only a grouped index.
	if err := os.RemoveAll(inspectorsOut); err != nil {
		return err
	}
	inspLayers := inspect.Layers()
	inspByLayer := map[string][]inspect.Descriptor{}
	for _, d := range inspect.Descriptors() {
		inspByLayer[d.Layer] = append(inspByLayer[d.Layer], d)
	}
	if err := write(filepath.Join(inspectorsOut, "_index.md"), inspectorsIndex(inspLayers, inspByLayer)); err != nil {
		return err
	}
	for li, layer := range inspLayers {
		dir := filepath.Join(inspectorsOut, layer.ID)
		ds := inspByLayer[layer.ID]
		if err := write(filepath.Join(dir, "_index.md"), inspectorLayerIndex(layer, ds, (li+1)*10)); err != nil {
			return err
		}
		for di, d := range ds {
			if err := write(filepath.Join(dir, d.Slug+".md"), inspectorPage(d, (di+1)*10)); err != nil {
				return err
			}
		}
	}

	// Worked examples: run each example and write the embeddable snippets the
	// reference, how-to, and deep-dive pages pull in. The feature examples are
	// embedded inline above (via workedExample); this writes their snippets.
	if err := prepareExamples(); err != nil {
		return err
	}

	// Governance pages: mirror the repo-root files into the contributing
	// section so the site carries them without duplicating their content by
	// hand. The root files remain the single source of truth.
	for _, g := range governanceDocs {
		page, err := governancePage(g)
		if err != nil {
			return err
		}
		if err := write(g.out, page); err != nil {
			return err
		}
	}
	return nil
}

// prepareExamples runs each worked example and writes its two embeddable
// snippets under examplesSnippetsOut: <id>.txt (raw command output, embedded by
// {{< katalyst-example >}}) and <id>.full.md (the full corpus+command+output at
// H3, embedded by {{< katalyst-example-full >}}). The same Run is gated by
// internal/examples' test, so the published output is a tested contract. It also
// removes the retired catalog section so it does not linger in the working tree.
func prepareExamples() error {
	if err := os.RemoveAll(examplesOut); err != nil {
		return err
	}
	if err := os.RemoveAll(examplesSnippetsOut); err != nil {
		return err
	}
	for _, ex := range examples.All() {
		res, err := examples.Run(ex)
		if err != nil {
			return fmt.Errorf("run example %s: %w", ex.ID, err)
		}
		if err := write(filepath.Join(examplesSnippetsOut, ex.ID+".txt"), examples.Output(res)); err != nil {
			return err
		}
		// H3 so the shortcode nests the example under a host page's
		// `## Worked example` heading.
		if err := write(filepath.Join(examplesSnippetsOut, ex.ID+".full.md"), examples.RenderPageAt(ex, res, 3)); err != nil {
			return err
		}
	}
	return nil
}

// governanceDoc mirrors a repo-root governance file into the docs site.
type governanceDoc struct {
	src    string // repo-root source, relative to the repo root
	out    string // generated page, relative to the repo root
	title  string // Hugo page title (matches the file's H1)
	weight int    // sort weight within the contributing section
}

// governanceDocs are the root files surfaced in docs/content/contributing/.
// GitHub requires them at the repo root for its community-health features, so
// the root files are canonical and these pages are generated from them.
var governanceDocs = []governanceDoc{
	{src: "CODE_OF_CONDUCT.md", out: "docs/content/contributing/code-of-conduct.md", title: "Code of conduct", weight: 70},
	{src: "SECURITY.md", out: "docs/content/contributing/security.md", title: "Security policy", weight: 80},
}

// governancePage wraps a root file's body in Hugo frontmatter and the
// generated-note banner. The body (including its H1) is copied verbatim; the
// root files only link to absolute URLs, so no relref rewriting is needed.
func governancePage(g governanceDoc) (string, error) {
	body, err := os.ReadFile(g.src)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "+++\ntitle = \"%s\"\nweight = %d\n+++\n\n", g.title, g.weight)
	fmt.Fprintf(&b, "<!-- GENERATED by cmd/gendocs from the repo-root %s. Do not edit by hand; run `make docs-gen`. -->\n\n", g.src)
	b.Write(body)
	return b.String(), nil
}

const inspectorsGeneratedNote = "<!-- GENERATED by cmd/gendocs from internal/inspect/registry.go. Do not edit by hand; run `make docs-gen`. -->"

func inspectorsIndex(layers []inspect.Layer, byLayer map[string][]inspect.Descriptor) string {
	var b strings.Builder
	fmt.Fprint(&b, "+++\ntitle = \"Inspectors\"\nweight = 45\nbookCollapseSection = true\n+++\n\n")
	fmt.Fprintln(&b, inspectorsGeneratedNote)
	fmt.Fprint(&b, "\n# Inspectors reference\n\n")
	fmt.Fprint(&b, "Inspectors describe the shape of content and return evidence: counts and ")
	fmt.Fprint(&b, "distributions, never recommendations. They are the descriptive dual of ")
	fmt.Fprint(&b, "[check types]({{< relref \"../check-types/_index.md\" >}}) and drive the ")
	fmt.Fprint(&b, "[`inspect`]({{< relref \"../cli.md\" >}}) command. They come in two layers: ")
	fmt.Fprint(&b, "raw base inspectors profile a base before configuration, collection ")
	fmt.Fprint(&b, "inspectors profile a configured collection. These pages are generated from the ")
	fmt.Fprint(&b, "inspector registry, so they always match the shipped binary.\n")
	for _, layer := range layers {
		ds := byLayer[layer.ID]
		if len(ds) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n\n%s\n\n", layer.Title, layer.Intro)
		for _, d := range ds {
			fmt.Fprintf(&b, "- [%s]({{< relref \"%s/%s.md\" >}}): %s\n", d.Title, layer.ID, d.Slug, plain(d.Summary))
		}
	}
	return b.String()
}

func inspectorLayerIndex(layer inspect.Layer, ds []inspect.Descriptor, weight int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "+++\ntitle = \"%s\"\nweight = %d\nbookCollapseSection = true\n+++\n\n", layer.Title, weight)
	fmt.Fprintln(&b, inspectorsGeneratedNote)
	fmt.Fprintf(&b, "\n%s\n\n", layer.Intro)
	fmt.Fprint(&b, "Inspectors in this layer:\n\n")
	for _, d := range ds {
		fmt.Fprintf(&b, "- [%s]({{< relref \"%s.md\" >}}): %s\n", d.Title, d.Slug, plain(d.Summary))
	}
	return b.String()
}

func inspectorPage(d inspect.Descriptor, weight int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "+++\ntitle = \"%s\"\nweight = %d\n+++\n\n", d.Title, weight)
	fmt.Fprintln(&b, inspectorsGeneratedNote)
	fmt.Fprintf(&b, "\n## Inspector ID\n\n`%s`\n\n", d.Name)
	fmt.Fprintf(&b, "## Layer\n\n%s\n\n", d.Layer)
	fmt.Fprintf(&b, "## Purpose\n\n%s\n\n", d.Summary)
	fmt.Fprint(&b, "## Usage\n\nInspectors emit evidence: counts and distributions, for the reader to ")
	fmt.Fprint(&b, "judge. Run this one with:\n\n")
	fmt.Fprintf(&b, "```\nkatalyst inspect <target> --inspector %s\n```\n", d.Name)
	b.WriteString(workedExample("inspector:" + d.Layer + "/" + d.Slug))
	return b.String()
}

const generatedNote = "<!-- GENERATED by cmd/gendocs from internal/checks/registry.go. Do not edit by hand; run `make docs-gen`. -->"

func sectionIndex(families []checks.Family, byFamily map[string][]checks.Descriptor) string {
	var b strings.Builder
	// aliases redirect the pre-rename /reference/rules/ URLs to this section.
	fmt.Fprint(&b, "+++\ntitle = \"Check types\"\nweight = 40\nbookCollapseSection = true\naliases = [\"/reference/rules/\"]\n+++\n\n")
	fmt.Fprintln(&b, generatedNote)
	fmt.Fprint(&b, "\n# Check types reference\n\n")
	fmt.Fprint(&b, "The check types `katalyst` runs against each item, grouped by family. ")
	fmt.Fprint(&b, "These pages are generated from the checks registry, so they always match the shipped binary.\n")
	for _, fam := range families {
		fmt.Fprintf(&b, "\n## %s\n\n%s\n\n", fam.Title, fam.Intro)
		for _, d := range byFamily[fam.ID] {
			fmt.Fprintf(&b, "- [%s]({{< relref \"%s/%s.md\" >}}): %s\n", d.Title, fam.Slug, d.Slug, plain(d.Summary))
		}
	}
	return b.String()
}

func familyIndex(fam checks.Family, ds []checks.Descriptor, weight int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "+++\ntitle = \"%s\"\nweight = %d\nbookCollapseSection = true\naliases = [\"/reference/rules/%s/\"]\n+++\n\n", fam.Title, weight, fam.Slug)
	fmt.Fprintln(&b, generatedNote)
	fmt.Fprintf(&b, "\n%s\n\n", fam.Intro)
	fmt.Fprint(&b, "Check types in this family:\n\n")
	for _, d := range ds {
		fmt.Fprintf(&b, "- [%s]({{< relref \"%s.md\" >}}): %s\n", d.Title, d.Slug, plain(d.Summary))
	}
	b.WriteString(workedExample("family:" + fam.Slug))
	return b.String()
}

func checkTypePage(d checks.Descriptor, fam checks.Family, weight int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "+++\ntitle = \"%s\"\nweight = %d\naliases = [\"/reference/rules/%s/%s/\"]\n+++\n\n", d.Title, weight, fam.Slug, d.Slug)
	fmt.Fprintln(&b, generatedNote)
	fmt.Fprintf(&b, "\n## Check type ID\n\n`kind: %s`\n\n", d.CheckType)
	if d.Scope == "collection" {
		fmt.Fprint(&b, "**Scope:** collection, runs once per collection over all its items.\n\n")
	}
	fmt.Fprintf(&b, "**Targets:** %s.\n\n", strings.Join(checks.DescriptorTargets(d), ", "))
	if d.Severity == "warning" {
		fmt.Fprint(&b, "**Severity:** warning, reported for review; never fails a run.\n\n")
	}
	fmt.Fprintf(&b, "## Purpose\n\n%s\n\n", d.Summary)
	if len(d.Fields) > 0 {
		fmt.Fprint(&b, "## Configuration keys\n\n")
		fmt.Fprint(&b, "| Field | Required | Default | Meaning |\n|---|---|---|---|\n")
		for _, f := range d.Fields {
			req := "no"
			if f.Required {
				req = "yes"
			}
			def := f.Default
			if def == "" {
				def = "-"
			} else {
				def = "`" + def + "`"
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", f.Name, req, def, f.Desc)
		}
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "## Example\n\n```yaml\n%s\n```\n", d.ConfigExample)
	b.WriteString(workedExample("checktype:" + fam.Slug + "/" + d.Slug))
	return b.String()
}

// plain strips inline-code backticks from a summary so it reads cleanly in a
// link list.
func plain(s string) string {
	return strings.ReplaceAll(s, "`", "")
}

// write creates parent directories and writes content with a trailing
// newline normalized.
func write(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out := []byte(content)
	if !bytes.HasSuffix(out, []byte("\n")) {
		out = append(out, '\n')
	}
	return os.WriteFile(path, out, 0o644)
}
