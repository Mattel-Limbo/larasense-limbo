package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/Mattel-Limbo/larasense-limbo/internal/fixer"
	"github.com/Mattel-Limbo/larasense-limbo/internal/reviewer"
)

func handleFixOutput(result *reviewer.Result, fix, apply, yes bool, patchPath string) error {
	if !fix {
		return nil
	}

	fixable := fixer.CountFixable(result.Issues)
	if fixable == 0 {
		log.Println("No issues with auto-fix data found.")
		return nil
	}

	log.Printf("Found %d issue(s) with auto-fix suggestions", fixable)

	// Write patch file if requested
	if patchPath != "" {
		if err := fixer.WritePatch(result.Issues, patchPath); err != nil {
			return fmt.Errorf("writing patch file: %w", err)
		}
		fmt.Printf("\n  📝 Patch written to %s (%d fix(es))\n", patchPath, fixable)
		fmt.Printf("     Apply with: git apply %s\n\n", patchPath)
	}

	// Apply fixes
	if apply {
		mode := fixer.ApplyInteractive
		if yes {
			mode = fixer.ApplyAll
		}

		applyResult, err := fixer.ApplyFixes(result.Issues, mode, os.Stdin)
		if err != nil {
			return fmt.Errorf("applying fixes: %w", err)
		}

		// Show applied
		if len(applyResult.Applied) > 0 {
			fmt.Printf("\n  🔧 Applied %d fix(es):\n", len(applyResult.Applied))
			for _, a := range applyResult.Applied {
				fmt.Printf("     ✅ %s:%d — %s\n", a.File, a.Line, a.Title)
			}
		}

		// Show skipped
		if len(applyResult.Skipped) > 0 {
			fmt.Printf("\n  ⏭️  Skipped %d fix(es):\n", len(applyResult.Skipped))
			for _, s := range applyResult.Skipped {
				fmt.Printf("     ⚠️  %s:%d — %s (%s)\n", s.File, s.Line, s.Title, s.Reason)
			}
		}

		if len(applyResult.Applied) == 0 && len(applyResult.Skipped) > 0 {
			fmt.Println("\n  💡 Tip: fixes were skipped because the code didn't match AI's suggestion.")
			fmt.Println("     Try running with --no-cache to get fresh fix data.")
		}

		fmt.Println()
	}

	return nil
}
