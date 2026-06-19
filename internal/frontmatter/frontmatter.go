// Package frontmatter extracts and parses YAML frontmatter from markdown
// documents.
//
// A frontmatter block is the YAML document enclosed between two "---"
// fences at the very top of a markdown file:
//
//	---
//	title: Dune
//	year: 1965
//	---
//	# Body starts here
//
// Only YAML frontmatter is supported in v0.1. TOML and JSON frontmatter
// are tracked in docs/contributing/roadmap.md.
package frontmatter

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Sentinel errors. Callers should use errors.Is to identify them rather
// than relying on string matching.
var (
	// ErrUnterminated indicates a file started with a "---" fence but
	// never had a matching closing fence.
	ErrUnterminated = errors.New("frontmatter: unterminated block (missing closing '---')")

	// ErrInvalidYAML indicates the bytes between the fences were not
	// valid YAML. The wrapped error contains the underlying yaml.v3
	// parser message.
	ErrInvalidYAML = errors.New("frontmatter: invalid YAML")
)

// Document is the result of parsing a markdown file.
//
// If HasFrontmatter is false, Meta is nil and Body holds the entire
// original input (verbatim).
//
// Lines maps JSON-pointer paths into Meta ("/title", "/tags/0", etc.)
// to their 1-indexed source line numbers in the original file. The
// opening "---" fence counts as line 1, so the first YAML key is
// typically at line 2. Lines is empty when HasFrontmatter is false.
type Document struct {
	HasFrontmatter bool
	Meta           map[string]any
	Body           []byte
	BodyLine       int
	Lines          map[string]int
}

// fence is the literal opener/closer for YAML frontmatter.
const fence = "---"

// bom is the UTF-8 byte-order mark, which some editors prepend to files.
var bom = []byte{0xEF, 0xBB, 0xBF}

// Parse extracts YAML frontmatter from src.
//
// The function never modifies src. It returns a Document whose Body is a
// sub-slice of src (after BOM stripping); callers that mutate the result
// should copy first.
func Parse(src []byte) (*Document, error) {
	input := bytes.TrimPrefix(src, bom)

	// Frontmatter only applies if the very first line is exactly the
	// fence. We require an immediate newline (or end-of-file) after the
	// fence so a line like "----" or "--- something" doesn't trigger.
	if !startsWithFence(input) {
		return &Document{Body: src, BodyLine: 1}, nil
	}

	// Skip past the opening fence and its line terminator.
	rest := input[len(fence):]
	rest = trimOneNewline(rest)

	// Find the closing fence on its own line.
	closeStart, closeEnd, ok := findClosingFence(rest)
	if !ok {
		return nil, ErrUnterminated
	}

	yamlBlock := rest[:closeStart]
	body := rest[closeEnd:]
	bodyLine := 1 + bytes.Count(input[:len(input)-len(rest)+closeEnd], []byte{'\n'})

	meta := map[string]any{}
	lines := map[string]int{}
	if len(bytes.TrimSpace(yamlBlock)) > 0 {
		var root yaml.Node
		if err := yaml.Unmarshal(yamlBlock, &root); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidYAML, err)
		}
		if err := root.Decode(&meta); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidYAML, err)
		}
		// The yaml.Node line numbers are 1-indexed relative to the
		// YAML block. We add 1 to account for the opening "---" fence
		// (which is line 1 of the original file).
		collectLines(&root, "", lines, 1)
	}

	return &Document{
		HasFrontmatter: true,
		Meta:           meta,
		Body:           body,
		BodyLine:       bodyLine,
		Lines:          lines,
	}, nil
}

// collectLines walks a yaml.Node tree and populates out with a mapping
// from JSON pointer paths to source line numbers. The offset is added
// to each node's reported line so callers can compensate for content
// that's embedded in a larger file (e.g. the leading "---" fence in
// markdown frontmatter).
func collectLines(n *yaml.Node, path string, out map[string]int, offset int) {
	if n == nil {
		return
	}
	switch n.Kind {
	case yaml.DocumentNode:
		for _, c := range n.Content {
			collectLines(c, path, out, offset)
		}
	case yaml.MappingNode:
		// Content alternates key, value, key, value, ... We index by
		// the key's line rather than the value's so that "tags:" on
		// line 4 reports /tags at line 4, not at line 5 (where the
		// first sequence item lives).
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i]
			val := n.Content[i+1]
			childPath := path + "/" + escapePointer(key.Value)
			out[childPath] = key.Line + offset
			collectLines(val, childPath, out, offset)
		}
	case yaml.SequenceNode:
		for i, c := range n.Content {
			childPath := fmt.Sprintf("%s/%d", path, i)
			out[childPath] = c.Line + offset
			collectLines(c, childPath, out, offset)
		}
	}
}

// escapePointer escapes a JSON pointer reference token per RFC 6901:
// "~" becomes "~0" and "/" becomes "~1". Frontmatter keys with these
// characters are rare but legal.
func escapePointer(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '~':
			out = append(out, '~', '0')
		case '/':
			out = append(out, '~', '1')
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}

// startsWithFence reports whether input begins with "---" followed
// immediately by a line terminator (or EOF).
func startsWithFence(input []byte) bool {
	if !bytes.HasPrefix(input, []byte(fence)) {
		return false
	}
	tail := input[len(fence):]
	if len(tail) == 0 {
		return true
	}
	return tail[0] == '\n' || tail[0] == '\r'
}

// trimOneNewline removes a single leading "\n" or "\r\n".
func trimOneNewline(b []byte) []byte {
	switch {
	case bytes.HasPrefix(b, []byte("\r\n")):
		return b[2:]
	case bytes.HasPrefix(b, []byte("\n")):
		return b[1:]
	default:
		return b
	}
}

// findClosingFence locates the next line that is exactly "---" (with no
// other content). It returns the byte offsets that delimit, respectively,
// the end of the YAML block and the start of the body.
//
// closeStart is the index in b where the closing-fence line begins (i.e.
// the start of "---"). closeEnd is the index of the first byte of the
// body — past the fence and its trailing newline (if any).
func findClosingFence(b []byte) (closeStart, closeEnd int, ok bool) {
	cursor := 0
	for cursor < len(b) {
		lineEnd := bytes.IndexByte(b[cursor:], '\n')
		var line []byte
		var lineTermLen int
		if lineEnd < 0 {
			line = b[cursor:]
			lineTermLen = 0
		} else {
			line = b[cursor : cursor+lineEnd]
			lineTermLen = 1
		}
		trimmed := bytes.TrimRight(line, "\r")
		if string(trimmed) == fence {
			return cursor, cursor + len(line) + lineTermLen, true
		}
		if lineEnd < 0 {
			break
		}
		cursor += lineEnd + 1
	}
	return 0, 0, false
}
