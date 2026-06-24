// Package skillpack packages the product skills under skills/ into .skill
// archives for distribution. A .skill is a zip of one skill directory with
// SKILL.md at the archive root (not nested under a {name}/ prefix — the client
// expects SKILL.md at the top), with the shared bootstrap copied in alongside
// it. Skills whose SKILL.md front matter is `status: placeholder` are not
// shippable and are excluded from packaging.
//
// The logic lives here, not in cmd/skillpack, so it is exercisable by an
// external test without shelling out to the built binary. `make skills` runs
// the cmd/skillpack wrapper over it.
package skillpack

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// BootstrapName is the filename the shared bootstrap takes inside each .skill.
const BootstrapName = "bootstrap.sh"

// Skill is one discovered skill directory.
type Skill struct {
	Name        string // directory name, == SKILL.md `name` (kept 1:1)
	Dir         string // absolute or relative path to the skill directory
	Placeholder bool   // SKILL.md front matter `status: placeholder`
}

// frontMatter is the subset of SKILL.md front matter packaging cares about.
type frontMatter struct {
	Name   string `yaml:"name"`
	Status string `yaml:"status"`
}

// Discover lists every skill directory under skillsDir (those containing a
// SKILL.md), reading each one's front matter to learn its name and whether it
// is a placeholder. The shared bootstrap (skillsDir/bootstrap.sh) is not a
// skill and is skipped.
func Discover(skillsDir string) ([]Skill, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, err
	}
	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(skillsDir, e.Name())
		fmPath := filepath.Join(dir, "SKILL.md")
		fm, err := readFrontMatter(fmPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // a directory without SKILL.md is not a skill
			}
			return nil, fmt.Errorf("%s: %w", fmPath, err)
		}
		skills = append(skills, Skill{
			Name:        e.Name(),
			Dir:         dir,
			Placeholder: strings.EqualFold(fm.Status, "placeholder"),
		})
	}
	return skills, nil
}

// readFrontMatter parses the leading `---` YAML block of a SKILL.md.
func readFrontMatter(path string) (frontMatter, error) {
	var fm frontMatter
	data, err := os.ReadFile(path)
	if err != nil {
		return fm, err
	}
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return fm, fmt.Errorf("missing front matter")
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return fm, fmt.Errorf("unterminated front matter")
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return fm, err
	}
	return fm, nil
}

// PackageAll packages every shippable skill under skillsDir into outDir,
// returning the artifact paths in directory order. Placeholders are skipped.
func PackageAll(skillsDir, outDir string) ([]string, error) {
	skills, err := Discover(skillsDir)
	if err != nil {
		return nil, err
	}
	bootstrap := filepath.Join(skillsDir, BootstrapName)
	var out []string
	for _, s := range skills {
		if s.Placeholder {
			continue
		}
		artifact, err := packageSkill(s, outDir, bootstrap)
		if err != nil {
			return nil, err
		}
		out = append(out, artifact)
	}
	return out, nil
}

// Package packages a single named skill into outDir, returning its artifact
// path. It errors if the skill does not exist or is a placeholder.
func Package(skillsDir, name, outDir string) (string, error) {
	skills, err := Discover(skillsDir)
	if err != nil {
		return "", err
	}
	for _, s := range skills {
		if s.Name != name {
			continue
		}
		if s.Placeholder {
			return "", fmt.Errorf("skill %q is a placeholder and is not shippable", name)
		}
		return packageSkill(s, outDir, filepath.Join(skillsDir, BootstrapName))
	}
	return "", fmt.Errorf("skill %q not found under %s", name, skillsDir)
}

// packageSkill zips the skill directory (SKILL.md and references/ at the
// archive root) plus the shared bootstrap into outDir/<name>.skill.
func packageSkill(s Skill, outDir, bootstrapPath string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	artifact := filepath.Join(outDir, s.Name+".skill")
	f, err := os.Create(artifact)
	if err != nil {
		return "", err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	err = filepath.Walk(s.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.Dir, path)
		if err != nil {
			return err
		}
		return addFile(zw, path, filepath.ToSlash(rel), info)
	})
	if err != nil {
		zw.Close()
		return "", err
	}

	// Copy the shared bootstrap in at the archive root, if present.
	if bi, err := os.Stat(bootstrapPath); err == nil {
		if err := addFile(zw, bootstrapPath, BootstrapName, bi); err != nil {
			zw.Close()
			return "", err
		}
	}

	if err := zw.Close(); err != nil {
		return "", err
	}
	return artifact, nil
}

// addFile writes one file into the zip at name, preserving its unix mode so the
// executable bit on scripts survives packaging.
func addFile(zw *zip.Writer, srcPath, name string, info os.FileInfo) error {
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = name
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = io.Copy(w, src)
	return err
}
