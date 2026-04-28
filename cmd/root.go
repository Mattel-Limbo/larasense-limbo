package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "larasense-limbo",
	Short: "AI-powered Laravel code reviewer",
	Long: `larasense-limbo analyzes git diffs in Laravel projects and uses an AI provider
to perform intelligent code review focused on Laravel best practices.

It checks for performance issues, security vulnerabilities, bad practices,
and Laravel convention violations in your changed code.`,
}

// Execute runs the root command.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
