// Command skillpack packages the product skills under skills/ into .skill
// archives (one zip per shippable skill, SKILL.md at the archive root, the
// shared bootstrap bundled in). Placeholders (`status: placeholder`) are
// skipped. Run via `make skills`; `make skill SKILL=<name>` packages one.
//
// The packaging logic lives in internal/skillpack so it can be tested without
// shelling out to this binary.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/abegong/katalyst/internal/skillpack"
)

func main() {
	skillsDir := flag.String("skills", "skills", "directory holding the skill folders")
	outDir := flag.String("out", "bin", "directory to write .skill artifacts into")
	one := flag.String("skill", "", "package only this skill (by directory name)")
	flag.Parse()

	if *one != "" {
		artifact, err := skillpack.Package(*skillsDir, *one, *outDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skillpack: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(artifact)
		return
	}

	artifacts, err := skillpack.PackageAll(*skillsDir, *outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skillpack: %v\n", err)
		os.Exit(1)
	}
	for _, a := range artifacts {
		fmt.Println(a)
	}
}
