package document

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Encode serializes a parsed document back into bytes in canonical form — the
// dual of Parse:
//
//   - the source format is preserved (TOML stays TOML, JSON stays JSON)
//   - top-level keys sorted alphabetically
//   - each format's default block/indent style
//   - exactly one trailing newline on the whole file
//   - body bytes preserved verbatim (leading blank lines trimmed)
//
// It assumes doc.HasFrontmatter; the no-frontmatter passthrough is the caller's
// policy (see internal/fix). Why this canonical form is intentionally inflexible
// is documented in the formatting deep-dive.
func Encode(doc *Document) ([]byte, error) {
	open, block, closeFence, err := marshalBlock(doc.Format, doc.Meta)
	if err != nil {
		return nil, fmt.Errorf("encode frontmatter: %w", err)
	}

	var out bytes.Buffer
	out.WriteString(open)
	out.Write(block)
	if !bytes.HasSuffix(block, []byte("\n")) {
		out.WriteByte('\n')
	}
	out.WriteString(closeFence)

	// The body starts immediately after the closing fence. Strip any further
	// leading blank lines (they were noise) and collapse trailing whitespace
	// into a single final newline.
	body := bytes.TrimLeft(doc.Body, "\n")
	body = bytes.TrimRight(body, "\n")
	out.Write(body)
	out.WriteByte('\n')

	return out.Bytes(), nil
}

// marshalBlock renders meta in the given format and returns the opening
// fence, the marshaled block (sans fences), and the closing fence.
func marshalBlock(format Kind, meta map[string]any) (open string, block []byte, closeFence string, err error) {
	switch format {
	case KindTOML:
		b, err := toml.Marshal(meta)
		if err != nil {
			return "", nil, "", err
		}
		return "+++\n", b, "+++\n", nil
	case KindJSON:
		b, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return "", nil, "", err
		}
		// MarshalIndent emits the braces; they are the JSON frontmatter's
		// fences, so there is nothing to add around the block.
		return "", b, "", nil
	default:
		b, err := marshalSortedYAML(meta)
		if err != nil {
			return "", nil, "", err
		}
		return "---\n", b, "---\n", nil
	}
}

// marshalSortedYAML emits a YAML mapping whose top-level keys are in
// alphabetical order. Nested map ordering follows yaml.v3's default
// (which is insertion order from the input).
func marshalSortedYAML(m map[string]any) ([]byte, error) {
	if len(m) == 0 {
		// yaml.v3 marshals an empty map as the literal "{}\n", which
		// is what we want.
		return yaml.Marshal(m)
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	root := &yaml.Node{Kind: yaml.MappingNode}
	for _, k := range keys {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: k}
		valNode := &yaml.Node{}
		if err := valNode.Encode(m[k]); err != nil {
			return nil, err
		}
		// Force block style on all containers so the output is the
		// "normal" multi-line YAML and not, e.g., a flow-style list.
		forceBlockStyle(valNode)
		root.Content = append(root.Content, keyNode, valNode)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	// yaml.NewEncoder.Encode for a single node sometimes prepends a
	// "---\n" document marker, which we add separately. Strip it if
	// present to avoid doubling.
	out := buf.Bytes()
	out = bytes.TrimPrefix(out, []byte("---\n"))
	return out, nil
}

// forceBlockStyle recursively clears any explicit flow-style hints,
// letting yaml.v3 emit the default block style.
func forceBlockStyle(n *yaml.Node) {
	if n == nil {
		return
	}
	if n.Style == yaml.FlowStyle {
		n.Style = 0
	}
	for _, c := range n.Content {
		forceBlockStyle(c)
	}
}
