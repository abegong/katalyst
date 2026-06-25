package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

const skillReleaseAPI = "https://api.github.com/repos/abegong/katalyst/releases/latest"
const skillReleaseDownloadBase = "https://github.com/abegong/katalyst/releases/download"

var shippedSkills = []string{
	"katalyst-overview",
	"katalyst-catalog",
	"katalyst-identify-collections",
	"katalyst-define-schemas",
	"katalyst-deploy",
	"katalyst-deploy-precommit-hook",
	"katalyst-deploy-cli-gating",
}

func newSkillsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "skills",
		Short: "Commands to list and install agent skills",
		Long: `skills manages the agent skill bundles that teach agents to use
katalyst. List the shipped skills, or download the latest release's .skill
bundles into a local directory so they can be imported into an agent client.`,
	}
	c.AddCommand(newSkillsListCmd(), newSkillsInstallCmd())
	return c
}

func newSkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List shipped agent skills",
		Args:  maxArgs(0, "skills list"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			printListSectionHeader(out, "Agent skills", len(shippedSkills))
			for _, skill := range shippedSkills {
				fmt.Fprintf(out, "- %s\n", skill)
			}
			return nil
		},
	}
}

func newSkillsInstallCmd() *cobra.Command {
	dir := "katalyst-skills"
	c := &cobra.Command{
		Use:   "install",
		Short: "Download latest agent skill bundles",
		Long: `install downloads the latest release's .skill bundles into a local
directory. Import those bundles into your agent client to enable the katalyst
agent workflow.`,
		Args: maxArgs(0, "skills install"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsInstall(cmd, dir)
		},
	}
	c.Flags().StringVar(&dir, "dir", dir, "Directory to write .skill bundles into.")
	return c
}

func runSkillsInstall(cmd *cobra.Command, dir string) error {
	if dir == "" {
		return usageErr("--dir: must not be empty")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	tag, err := latestReleaseTag(client)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Downloading katalyst agent skills from %s\n", tag)
	for _, skill := range shippedSkills {
		name := skill + ".skill"
		url := fmt.Sprintf("%s/%s/%s", skillReleaseDownloadBase, tag, name)
		dst := filepath.Join(dir, name)
		if err := downloadFile(client, url, dst); err != nil {
			return err
		}
		fmt.Fprintf(out, "- %s\n", dst)
	}
	fmt.Fprintf(out, "\nImport these .skill files into your agent client.\n")
	return nil
}

func latestReleaseTag(client *http.Client) (string, error) {
	req, err := http.NewRequest(http.MethodGet, skillReleaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch latest release: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("parse latest release: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("parse latest release: missing tag_name")
	}
	return release.TagName, nil
}

func downloadFile(client *http.Client, url, dst string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}
