package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	"github.com/tyuhara/yamldiff/internal/config"
	"github.com/tyuhara/yamldiff/internal/diff"
	"github.com/tyuhara/yamldiff/internal/github"
	"github.com/tyuhara/yamldiff/internal/parser"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type CLI struct {
	// Global flags
	Version kong.VersionFlag `help:"Show version information."`

	// Subcommands
	Compare CompareCmd `cmd:"" help:"Compare two YAML files." default:"withargs"`
}

type CompareCmd struct {
	File1      string `arg:"" help:"First YAML file to compare." type:"existingfile"`
	File2      string `arg:"" help:"Second YAML file to compare." type:"existingfile"`
	Key        string `help:"YAML path to use as document identifier." default:"metadata.name"`
	ShowCounts bool   `short:"c" help:"Show summary counts only."`
	Verbose    bool   `short:"v" help:"Show verbose output with full document content."`
	NoColor    bool   `help:"Disable color output."`

	// GitHub integration (legacy flags)
	GithubLabel    bool   `help:"Add GitHub label based on diff results."`
	GithubRepo     string `help:"GitHub repository (owner/repo). Required with --github-label."`
	GithubPR       int    `help:"GitHub PR number. Required with --github-label."`
	GithubToken    string `help:"GitHub token (or use GITHUB_TOKEN env var)."`
	ChangesLabel   string `help:"Label to add when changes are found." default:"config-sync/changes"`
	NoChangesLabel string `help:"Label to add when no changes are found." default:"config-sync/no-changes"`

	// Config file (tfcmt-style)
	Config      string            `help:"Path to yamldiff.yaml config file." type:"existingfile"`
	PostComment bool              `help:"Post comment to GitHub PR (requires --config)."`
	Link        string            `help:"CI build link to include in comment."`
	Var         map[string]string `help:"Variables to pass to template (key=value)."`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Description("Compare YAML files with support for multiple documents"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": fmt.Sprintf("%s (commit: %s, built at: %s)", version, commit, date),
		},
	)

	err := ctx.Run(&cli)
	ctx.FatalIfErrorf(err)
}

func (c *CompareCmd) Run(cli *CLI) error {
	// Disable color if requested
	if c.NoColor {
		color.NoColor = true
	}

	// Parse both files
	docs1, err := parser.ParseMultiDocYAML(c.File1)
	if err != nil {
		return fmt.Errorf("error parsing %s: %w", c.File1, err)
	}

	docs2, err := parser.ParseMultiDocYAML(c.File2)
	if err != nil {
		return fmt.Errorf("error parsing %s: %w", c.File2, err)
	}

	// Create diff engine
	engine := diff.NewEngine(c.Key)

	// Compare documents
	result := engine.Compare(docs1, docs2)

	// Capture detailed output for comment/template
	var detailsBuf bytes.Buffer
	if c.Verbose {
		// Use verbose output for details
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result.Print(true)

		w.Close()
		os.Stdout = oldStdout

		buf := new(bytes.Buffer)
		buf.ReadFrom(r)
		detailsBuf = *buf
	}

	// Print results to stdout (unless only posting comment)
	if !c.PostComment || c.Config == "" {
		if c.ShowCounts {
			result.PrintSummary()
		} else {
			result.Print(c.Verbose)
		}
	}

	// Handle config file-based GitHub integration
	if c.Config != "" {
		if err := c.handleConfigBasedIntegration(result, detailsBuf.String()); err != nil {
			return err
		}
	} else if c.GithubLabel {
		// Legacy GitHub labeling
		if err := c.applyGithubLabel(result); err != nil {
			return fmt.Errorf("error applying GitHub label: %w", err)
		}
	}

	// Exit with non-zero if differences found
	if result.HasDifferences() {
		os.Exit(1)
	}

	return nil
}

func (c *CompareCmd) handleConfigBasedIntegration(result *diff.Result, details string) error {
	// Load config file
	cfg, err := config.LoadConfig(c.Config)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Determine repo and PR number
	repo := cfg.GetRepoFullName()
	if c.GithubRepo != "" {
		repo = c.GithubRepo
	}
	prNumber := c.GithubPR

	if repo == "" {
		return fmt.Errorf("repository not specified in config or --github-repo")
	}
	if prNumber == 0 {
		return fmt.Errorf("PR number not specified (use --github-pr)")
	}

	// Validate token
	token := c.GithubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return fmt.Errorf("GitHub token not provided (use --github-token or GITHUB_TOKEN env var)")
	}

	compareConfig := cfg.YAMLDiff.Compare

	// Post comment if requested and template is configured
	if c.PostComment && !compareConfig.DisableComment && compareConfig.Template != "" {
		// Prepare template variables
		vars := make(map[string]interface{})
		for k, v := range c.Var {
			vars[k] = v
		}

		// Prepare template data
		templateData := github.PrepareTemplateData(result, details, c.Link, vars)

		// Render template
		commentBody, err := github.RenderTemplate(compareConfig.Template, templateData)
		if err != nil {
			return fmt.Errorf("error rendering template: %w", err)
		}

		// Post comment
		if err := github.PostComment(repo, prNumber, commentBody); err != nil {
			return fmt.Errorf("error posting comment: %w", err)
		}
	}

	// Add label if not disabled
	if !compareConfig.DisableLabel {
		labels := compareConfig.GetLabels(len(result.Added), len(result.Deleted), len(result.Modified))
		if len(labels) > 0 {
			if err := github.AddLabels(repo, prNumber, labels); err != nil {
				return fmt.Errorf("error adding labels: %w", err)
			}
		}
	}

	return nil
}

func (c *CompareCmd) applyGithubLabel(result *diff.Result) error {
	// Validate required parameters
	if c.GithubRepo == "" {
		return fmt.Errorf("--github-repo is required when using --github-label")
	}
	if c.GithubPR == 0 {
		return fmt.Errorf("--github-pr is required when using --github-label")
	}

	// Get token from flag or environment
	token := c.GithubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return fmt.Errorf("GitHub token not provided (use --github-token or GITHUB_TOKEN env var)")
	}

	// Determine which label to apply
	label := c.NoChangesLabel
	if result.HasDifferences() {
		label = c.ChangesLabel
	}

	// Apply label using gh CLI
	return applyGithubLabelWithGH(c.GithubRepo, c.GithubPR, label)
}

func applyGithubLabelWithGH(repo string, prNumber int, label string) error {
	cmd := exec.Command("gh", "pr", "edit", fmt.Sprintf("%d", prNumber),
		"--repo", repo,
		"--add-label", label)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add label: %w\nOutput: %s", err, string(output))
	}

	fmt.Fprintf(os.Stderr, "✓ Applied GitHub label: %s\n", label)
	return nil
}
