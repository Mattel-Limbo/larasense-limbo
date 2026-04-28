package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const configTemplate = `provider:
  base_url: https://api.openai.com
  api_key: ${AI_API_KEY}
  model: gpt-4.1
  endpoint: chat  # "chat" for /v1/chat/completions, "responses" for /v1/responses

review:
  max_issues: 5
  severity_threshold: medium

filters:
  include:
    - "app/**"
    - "routes/**"
    - "resources/views/**"
  exclude:
    - "tests/**"
    - "database/seeders/**"
`

var flagForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a .larasense-limbo.yml config file",
	Long: `Generate a default .larasense-limbo.yml configuration file in the current directory.

This creates a starter config with sensible defaults that you can customize
for your Laravel project and AI provider.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&flagForce, "force", false, "Overwrite existing config file")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	const configFile = ".larasense-limbo.yml"

	// Check if file already exists
	if _, err := os.Stat(configFile); err == nil && !flagForce {
		return fmt.Errorf("%s already exists (use --force to overwrite)", configFile)
	}

	if err := os.WriteFile(configFile, []byte(configTemplate), 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	fmt.Printf("Created %s\n", configFile)
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit .larasense-limbo.yml — set your provider base_url, api_key, and model")
	fmt.Println("  2. Add .larasense-limbo.yml to .gitignore (if it contains secrets)")
	fmt.Println("  3. Run: larasense-limbo analyze")

	return nil
}
