package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure bbranch with your Bitbucket credentials",
	Long:  "Interactive setup that prompts for API token credentials and writes .bbranch.yaml.",
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	var (
		workspace    string
		email        string
		token        string
		sourceBranch string
	)

	sourceBranch = "master"

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Workspace slug").
				Description("Your Bitbucket workspace identifier").
				Value(&workspace).
				Validate(requiredValidator("workspace")),
			huh.NewInput().
				Title("Bitbucket email").
				Description("Email associated with your API token").
				Value(&email).
				Validate(requiredValidator("email")),
			huh.NewInput().
				Title("API token").
				Description("Create at: Bitbucket > Personal settings > App passwords").
				EchoMode(huh.EchoModePassword).
				Value(&token).
				Validate(requiredValidator("API token")),
			huh.NewInput().
				Title("Default source branch").
				Value(&sourceBranch),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("setup cancelled")
	}

	configPath := ".bbranch.yaml"

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		var overwrite bool
		confirm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(".bbranch.yaml already exists. Overwrite?").
					Value(&overwrite),
			),
		)
		if err := confirm.Run(); err != nil {
			return fmt.Errorf("setup cancelled")
		}
		if !overwrite {
			fmt.Println("Setup cancelled — existing config preserved.")
			return nil
		}
	}

	// Default to "master" if user cleared the field
	if sourceBranch == "" {
		sourceBranch = "master"
	}

	// Use %q to safely quote values that may contain YAML special characters
	content := fmt.Sprintf(`workspace: %q

api_token:
  email: %q
  token: %q

defaults:
  source_branch: %q
`, workspace, email, token, sourceBranch)

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen, color.Bold)

	green.Println("✓ Configuration saved to .bbranch.yaml")
	fmt.Println()
	bold.Println("Next steps:")
	fmt.Println("  bbranch list              — list workspace repos")
	fmt.Println("  bbranch create <branch>   — create a branch across repos")

	return nil
}

func requiredValidator(field string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
}
