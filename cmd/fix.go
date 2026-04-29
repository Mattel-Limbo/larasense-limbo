package cmd

import (
	"fmt"
	"log"

	"github.com/Mattel-Limbo/larasense-limbo/internal/fixer"
	"github.com/Mattel-Limbo/larasense-limbo/internal/reviewer"
)

func handleFixOutput(result *reviewer.Result, fix, apply bool, patchPath string) error {
	if !fix {
		return nil
	}

	fixable := fixer.CountFixable(result.Issues)
	if fixable == 0 {
		return nil
	}

	log.Printf("Found %d issue(s) with auto-fix suggestions", fixable)

	if patchPath != "" {
		if err := fixer.WritePatch(result.Issues, patchPath); err != nil {
			return fmt.Errorf("writing patch file: %w", err)
		}
		fmt.Printf("\n  📝 Patch written to %s (%d fix(es))\n", patchPath, fixable)
		fmt.Printf("     Apply with: git apply %s\n\n", patchPath)
	}

	if apply {
		applied, err := fixer.ApplyFixes(result.Issues)
		if err != nil {
			return fmt.Errorf("applying fixes: %w", err)
		}
		fmt.Printf("\n  🔧 Applied %d fix(es):\n", len(applied))
		for _, a := range applied {
			fmt.Printf("     ✅ %s:%d — %s\n", a.File, a.Line, a.Title)
		}
		fmt.Println()
	}

	return nil
}
