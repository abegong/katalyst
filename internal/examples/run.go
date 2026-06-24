package examples

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/cmd"
)

// Result is the captured outcome of running an Example.
type Result struct {
	Stdout string
	Stderr string
	// Exit is the process exit code: 0 ok, 1 violations, 2 usage.
	Exit int
	// After holds the post-run content of each ResultFiles entry, normalized.
	After map[string]string
}

// Run scaffolds the example's corpus in a fresh temp directory, runs the
// `katalyst` command there via cmd.NewRootCmd, and returns the captured output.
// Absolute temp paths in the output are normalized to "<project>" so the result
// is deterministic. The returned error is only for infrastructure failures
// (writing the corpus, chdir); a non-zero command exit is reported in
// Result.Exit, not as an error.
func Run(ex Example) (Result, error) {
	dir, err := os.MkdirTemp("", "katalyst-example-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(dir)

	for _, f := range ex.Files {
		p := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return Result{}, err
		}
		if err := os.WriteFile(p, []byte(f.Content), 0o644); err != nil {
			return Result{}, err
		}
	}

	prev, err := os.Getwd()
	if err != nil {
		return Result{}, err
	}
	if err := os.Chdir(dir); err != nil {
		return Result{}, err
	}
	defer func() { _ = os.Chdir(prev) }()

	root := cmd.NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(ex.Args)
	runErr := root.Execute()

	// Resolve the temp dir through the same symlink evaluation the config
	// loader applies, so the normalization catches the form that appears in
	// diagnostics (on macOS /tmp lives behind /private/var).
	norm := func(s string) string {
		s = strings.ReplaceAll(s, dir, "<project>")
		if resolved, e := filepath.EvalSymlinks(dir); e == nil {
			s = strings.ReplaceAll(s, resolved, "<project>")
		}
		return s
	}

	res := Result{
		Stdout: norm(outBuf.String()),
		Stderr: norm(errBuf.String()),
		Exit:   exitCode(runErr),
	}
	if len(ex.ResultFiles) > 0 {
		res.After = map[string]string{}
		for _, rel := range ex.ResultFiles {
			b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
			if err != nil {
				return Result{}, err
			}
			res.After[rel] = norm(string(b))
		}
	}
	return res, nil
}

// exitCode maps a command error to its process exit code, following the
// cmd package's coded-error contract (cmd/check.go).
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ Code() int }
	if errors.As(err, &coded) {
		return coded.Code()
	}
	return 1
}

// Output returns what the command printed: stdout, then stderr, then an
// `exit status N` line when the exit code is non-zero. This is the snippet
// embedded into prose docs via the {{< katalyst-example >}} shortcode.
func Output(r Result) string {
	var b strings.Builder
	b.WriteString(r.Stdout)
	if r.Stderr != "" {
		ensureNL(&b)
		b.WriteString(r.Stderr)
	}
	if r.Exit != 0 {
		ensureNL(&b)
		fmt.Fprintf(&b, "exit status %d\n", r.Exit)
	}
	return b.String()
}

// RenderPage returns the Markdown body of an example's catalog page: the
// narrative, the input corpus, the command transcript, and any resulting files.
// cmd/gendocs wraps this in Hugo frontmatter and the generated-note banner; the
// test snapshots it directly. The output is deterministic.
func RenderPage(ex Example, res Result) string {
	var b strings.Builder
	if ex.Doc != "" {
		b.WriteString(ex.Doc)
		b.WriteString("\n\n")
	}

	b.WriteString("## Input\n\n")
	for _, f := range ex.Files {
		fmt.Fprintf(&b, "`%s`\n\n", f.Path)
		fence(&b, lang(f.Path), f.Content)
	}

	b.WriteString("## Command\n\n")
	var t strings.Builder
	fmt.Fprintf(&t, "$ katalyst %s\n", shellArgs(ex.Args))
	t.WriteString(Output(res))
	fence(&b, "console", t.String())

	if len(ex.ResultFiles) > 0 {
		b.WriteString("## Result\n\n")
		for _, rel := range ex.ResultFiles {
			fmt.Fprintf(&b, "`%s` after `katalyst %s`:\n\n", rel, shellArgs(ex.Args))
			fence(&b, lang(rel), res.After[rel])
		}
	}
	return b.String()
}

// fence writes a fenced code block, ensuring the body ends in a newline.
func fence(b *strings.Builder, language, body string) {
	b.WriteString("```")
	b.WriteString(language)
	b.WriteString("\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("```\n\n")
}

// ensureNL appends a newline if the builder does not already end with one.
func ensureNL(b *strings.Builder) {
	s := b.String()
	if s != "" && !strings.HasSuffix(s, "\n") {
		b.WriteString("\n")
	}
}

// lang maps a corpus file's extension to a fenced-code language hint.
func lang(path string) string {
	switch filepath.Ext(path) {
	case ".md":
		return "markdown"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	default:
		return "text"
	}
}

// shellArgs joins command arguments for display, quoting any that contain
// whitespace.
func shellArgs(args []string) string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t") {
			out[i] = fmt.Sprintf("%q", a)
		} else {
			out[i] = a
		}
	}
	return strings.Join(out, " ")
}
