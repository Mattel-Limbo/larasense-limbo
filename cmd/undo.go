package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mattel-Limbo/larasense-limbo/internal/fixer"
)

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Revert all applied fixes to their original state",
	Long: `Undo all fixes that were applied by --fix --apply.

Restores files from the backup created before fixes were applied.
The backup is removed after a successful undo.`,
	RunE: runUndo,
}

func init() {
	rootCmd.AddCommand(undoCmd)
}

func runUndo(cmd *cobra.Command, args []string) error {
	if !fixer.HasBackup() {
		return fmt.Errorf("no backup found — nothing to undo (fixes may have already been undone or were never applied)")
	}

	restored, err := fixer.UndoFixes()
	if err != nil {
		return fmt.Errorf("undo failed: %w", err)
	}

	fmt.Printf("\n  ↩️  Restored %d file(s) to original state:\n", len(restored))
	for _, f := range restored {
		fmt.Printf("     ✅ %s\n", f)
	}
	fmt.Println("\n  Backup removed. Files are back to their pre-fix state.")
	fmt.Println()

	return nil
}
