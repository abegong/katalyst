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
// are tracked in product/roadmap.md.
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
type Document struct {
	HasFrontmatter bool
	Meta           map[string]any
	Body           []byte
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
		return &Document{Body: src}, nil
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

	meta := map[string]any{}
	if len(bytes.TrimSpace(yamlBlock)) > 0 {
		if err := yaml.Unmarshal(yamlBlock, &meta); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidYAML, err)
		}
	}

	return &Document{
		HasFrontmatter: true,
		Meta:           meta,
		Body:           body,
	}, nil
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
