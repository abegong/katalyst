package frontmatter

import (
	"bytes"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// Format normalizes a markdown document's frontmatter:
//
//   - top-level keys sorted alphabetically
//   - yaml.v3 default block style
//   - exactly one trailing newline on the whole file
//   - body bytes preserved verbatim
//
// Files without frontmatter are returned unchanged. See
// docs/explanation/formatting.md for why this is intentionally inflexible.
func Format(src []byte) ([]byte, error) {
	doc, err := Parse(src)
	if err != nil {
		return nil, err
	}
	if !doc.HasFrontmatter {
		return src, nil
	}

	yamlBytes, err := marshalSorted(doc.Meta)
	if err != nil {
		return nil, fmt.Errorf("format frontmatter: %w", err)
	}

	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(yamlBytes)
	if !bytes.HasSuffix(yamlBytes, []byte("\n")) {
		out.WriteByte('\n')
	}
	out.WriteString("---\n")

	// The body returned by Parse starts immediately after the closing
	// "---\n" terminator. Strip any further leading blank lines (they
	// were noise) and collapse trailing whitespace into a single
	// final newline.
	body := bytes.TrimLeft(doc.Body, "\n")
	body = bytes.TrimRight(body, "\n")
	out.Write(body)
	out.WriteByte('\n')

	return out.Bytes(), nil
}

// marshalSorted emits a YAML mapping whose top-level keys are in
// alphabetical order. Nested map ordering follows yaml.v3's default
// (which is insertion order from the input).
func marshalSorted(m map[string]any) ([]byte, error) {
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
