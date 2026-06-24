// Package fix is the transform engine for `katalyst fix`: given an item's
// content and its collection, it computes the canonical, fixed form. It is
// backend-agnostic and does no file IO — persisting the result is the storage
// backend's job (see internal/storage/collection/filesystem). The output is
// byte-for-byte what the previous in-cmd implementation produced.
package fix

import (
	"fmt"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/plaintext"
	"github.com/abegong/katalyst/internal/project/config"
	"github.com/abegong/katalyst/internal/storage/collection/document"
)

// Apply returns the canonical, fixed form of src for collection c: it applies
// the collection's opted-in text_forbids body fixes, then rewrites the
// frontmatter into canonical form. The result equals src when nothing changes.
func Apply(src []byte, c config.Collection) ([]byte, error) {
	fixed, err := applyTextFixes(src, c)
	if err != nil {
		return nil, err
	}
	return Canonical(fixed)
}

// Canonical rewrites a markdown document's frontmatter into canonical form
// (top-level keys sorted, default block style, exactly one trailing newline,
// body verbatim). Files without frontmatter are returned unchanged. It composes
// the document codec's Parse and Encode.
func Canonical(src []byte) ([]byte, error) {
	doc, err := document.Parse(src)
	if err != nil {
		return nil, err
	}
	if !doc.HasFrontmatter {
		return src, nil
	}
	return document.Encode(doc)
}

// applyTextFixes rewrites the body with the collection's opted-in text_forbids
// fixes, then re-checks its own work: if a fix leaves the rule still violated
// (a bad template), it fails rather than producing a still-broken result. Files
// in collections with no text fixes are returned untouched.
func applyTextFixes(src []byte, c config.Collection) ([]byte, error) {
	fixers := textFixers(c)
	if len(fixers) == 0 {
		return src, nil
	}
	doc, err := document.Parse(src)
	if err != nil {
		return nil, err
	}
	body := doc.Body
	for _, f := range fixers {
		body = f.ApplyFix(body)
	}
	rechecked := &document.Document{Body: body, BodyLine: doc.BodyLine}
	for _, f := range fixers {
		if len(f.Run(checks.Context{Doc: rechecked})) > 0 {
			return nil, fmt.Errorf("fix did not resolve the violation for /%s/", f.Pattern)
		}
	}
	// Body is a verbatim tail of src, so everything before it is the prefix.
	prefix := src[:len(src)-len(doc.Body)]
	out := make([]byte, 0, len(prefix)+len(body))
	out = append(out, prefix...)
	out = append(out, body...)
	return out, nil
}

// textFixers builds the fixable text_forbids checks configured for a
// collection (those with a non-empty fix template). Each check is built from its
// validated config through the registry, so fix reuses the same TextForbids the
// engine would run.
func textFixers(c config.Collection) []plaintext.TextForbids {
	var out []plaintext.TextForbids
	for _, cc := range c.Checks {
		if cc.Kind != checks.CheckTextForbids {
			continue
		}
		chk, ok := checks.Build(cc.Kind, cc.Args)
		if !ok {
			continue
		}
		if tf, ok := chk.(plaintext.TextForbids); ok && tf.Fix != "" {
			out = append(out, tf)
		}
	}
	return out
}
