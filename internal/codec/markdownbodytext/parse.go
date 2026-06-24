package markdownbodytext

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Sentinel errors. Callers should use errors.Is to identify them rather
// than relying on string matching.
var (
	// ErrUnterminated indicates a file started with a frontmatter fence
	// but never had a matching closing fence.
	ErrUnterminated = errors.New("frontmatter: unterminated block")

	// ErrInvalidYAML indicates the bytes between the fences were not
	// valid YAML. The wrapped error contains the underlying yaml.v3
	// parser message.
	ErrInvalidYAML = errors.New("frontmatter: invalid YAML")

	// ErrInvalidTOML indicates the bytes between the "+++" fences were
	// not valid TOML.
	ErrInvalidTOML = errors.New("frontmatter: invalid TOML")

	// ErrInvalidJSON indicates the bytes between the "{" / "}" fences
	// were not a valid JSON object.
	ErrInvalidJSON = errors.New("frontmatter: invalid JSON")
)

// Kind identifies which frontmatter syntax a document uses. (It is named
// Kind rather than Format to avoid colliding with the Format function.)
type Kind int

const (
	// KindYAML is "---"-fenced YAML. It is also the zero value, used for
	// documents that have no frontmatter at all (where it is moot).
	KindYAML Kind = iota
	// KindTOML is "+++"-fenced TOML.
	KindTOML
	// KindJSON is "{" / "}"-delimited JSON.
	KindJSON
)

// String returns the lowercase name of the format ("yaml", "toml", "json").
func (k Kind) String() string {
	switch k {
	case KindTOML:
		return "toml"
	case KindJSON:
		return "json"
	default:
		return "yaml"
	}
}

// Document is the result of parsing a markdown file.
//
// If HasFrontmatter is false, Meta is nil and Body holds the entire
// original input (verbatim).
//
// Lines maps JSON-pointer paths into Meta ("/title", "/tags/0", etc.)
// to their 1-indexed source line numbers in the original file. The
// opening fence counts as line 1, so the first key is typically at line
// 2. Lines is empty when HasFrontmatter is false, and (today) for TOML
// and JSON frontmatter.
type Document struct {
	HasFrontmatter bool
	// Format records the source syntax so the formatter can preserve it.
	Format   Kind
	Meta     map[string]any
	Body     []byte
	BodyLine int
	Lines    map[string]int
	// Frontmatter is the raw metadata block, for text search; Meta is
	// its parsed form. For YAML and TOML this is the bytes between the
	// fences (no fences). For JSON it is the object including its braces,
	// since the braces are part of the JSON. Nil when HasFrontmatter is
	// false.
	Frontmatter []byte
}

// Fence literals for the line-fenced formats.
const (
	fenceYAML = "---"
	fenceTOML = "+++"
)

// bom is the UTF-8 byte-order mark, which some editors prepend to files.
var bom = []byte{0xEF, 0xBB, 0xBF}

// Parse extracts frontmatter from src, detecting the format (YAML, TOML,
// or JSON) from the opening fence.
//
// The function never modifies src. It returns a Document whose Body is a
// sub-slice of src (after BOM stripping); callers that mutate the result
// should copy first.
func Parse(src []byte) (*Document, error) {
	input := bytes.TrimPrefix(src, bom)

	switch {
	case startsWithFence(input, fenceYAML):
		return parseFenced(input, KindYAML, fenceYAML, decodeYAML)
	case startsWithFence(input, fenceTOML):
		return parseFenced(input, KindTOML, fenceTOML, decodeTOML)
	case len(input) > 0 && input[0] == '{':
		return parseJSON(input)
	default:
		// No frontmatter: Body is the original input, verbatim.
		return &Document{Body: src, BodyLine: 1}, nil
	}
}

// decoder turns a raw frontmatter block into parsed metadata and a
// (possibly empty) path→line index. offset is added to every reported
// line so the decoder can compensate for the opening fence on line 1.
type decoder func(block []byte, offset int) (map[string]any, map[string]int, error)

// parseFenced handles the line-fenced formats (YAML "---" and TOML
// "+++"), which share the same opening/closing structure and differ only
// in their decoder.
func parseFenced(input []byte, format Kind, fence string, decode decoder) (*Document, error) {
	// Skip past the opening fence and its line terminator.
	rest := input[len(fence):]
	rest = trimOneNewline(rest)

	closeStart, closeEnd, ok := findClosingFence(rest, fence)
	if !ok {
		return nil, fmt.Errorf("%w: missing closing %q", ErrUnterminated, fence)
	}

	block := rest[:closeStart]
	body := rest[closeEnd:]
	bodyLine := 1 + bytes.Count(input[:len(input)-len(rest)+closeEnd], []byte{'\n'})

	meta, lines, err := decode(block, 1)
	if err != nil {
		return nil, err
	}

	return &Document{
		HasFrontmatter: true,
		Format:         format,
		Meta:           meta,
		Body:           body,
		BodyLine:       bodyLine,
		Lines:          lines,
		Frontmatter:    block,
	}, nil
}

// parseJSON handles JSON frontmatter, which is a single brace-delimited
// object at the top of the file rather than a pair of fence lines.
func parseJSON(input []byte) (*Document, error) {
	end, ok := findJSONObjectEnd(input)
	if !ok {
		return nil, fmt.Errorf("%w: missing closing %q", ErrUnterminated, "}")
	}

	block := input[:end+1]
	rest := input[end+1:]
	afterFence := trimOneNewline(rest)
	consumed := len(input) - len(afterFence)
	bodyLine := 1 + bytes.Count(input[:consumed], []byte{'\n'})

	meta, lines, err := decodeJSON(block, 1)
	if err != nil {
		return nil, err
	}

	return &Document{
		HasFrontmatter: true,
		Format:         KindJSON,
		Meta:           meta,
		Body:           afterFence,
		BodyLine:       bodyLine,
		Lines:          lines,
		Frontmatter:    block,
	}, nil
}

// decodeYAML parses a YAML block and builds a full path→line index using
// the yaml.v3 node tree.
func decodeYAML(block []byte, offset int) (map[string]any, map[string]int, error) {
	meta := map[string]any{}
	lines := map[string]int{}
	if len(bytes.TrimSpace(block)) == 0 {
		return meta, lines, nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(block, &root); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}
	if err := root.Decode(&meta); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}
	collectLines(&root, "", lines, offset)
	return meta, lines, nil
}

// decodeTOML parses a TOML block. Line tracking is not yet implemented
// for TOML; Lines is returned empty.
func decodeTOML(block []byte, _ int) (map[string]any, map[string]int, error) {
	meta := map[string]any{}
	lines := map[string]int{}
	if len(bytes.TrimSpace(block)) == 0 {
		return meta, lines, nil
	}
	if err := toml.Unmarshal(block, &meta); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidTOML, err)
	}
	return meta, lines, nil
}

// decodeJSON parses a JSON object block. Line tracking is not yet
// implemented for JSON; Lines is returned empty.
func decodeJSON(block []byte, _ int) (map[string]any, map[string]int, error) {
	meta := map[string]any{}
	lines := map[string]int{}
	if len(bytes.TrimSpace(block)) == 0 {
		return meta, lines, nil
	}
	if err := json.Unmarshal(block, &meta); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	return meta, lines, nil
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

// startsWithFence reports whether input begins with fence followed
// immediately by a line terminator (or EOF).
func startsWithFence(input []byte, fence string) bool {
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

// findClosingFence locates the next line that is exactly fence (with no
// other content). It returns the byte offsets that delimit, respectively,
// the end of the metadata block and the start of the body.
//
// closeStart is the index in b where the closing-fence line begins.
// closeEnd is the index of the first byte of the body, past the fence
// and its trailing newline (if any).
func findClosingFence(b []byte, fence string) (closeStart, closeEnd int, ok bool) {
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

// findJSONObjectEnd returns the index of the "}" that closes the leading
// JSON object in b (which must begin with "{"). It tracks brace depth and
// skips over string literals so that braces inside strings don't count.
func findJSONObjectEnd(b []byte) (end int, ok bool) {
	depth := 0
	inStr := false
	esc := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}
