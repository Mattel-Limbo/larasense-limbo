package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Mattel-Limbo/larasense-limbo/internal/fixer"
	"github.com/Mattel-Limbo/larasense-limbo/internal/git"
	"github.com/Mattel-Limbo/larasense-limbo/internal/reviewer"
)

func handleFixOutput(result *reviewer.Result, fix, apply, yes, dryRun bool, gitBranch, patchPath string) error {
	if !fix {
		return nil
	}

	fixable := fixer.CountFixable(result.Issues)
	if fixable == 0 {
		log.Println("No issues with auto-fix data found.")
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

	if dryRun {
		fmt.Print("\n  🔍 Dry-run mode — no files will be modified\n\n")
		applyResult, err := fixer.ApplyFixes(result.Issues, fixer.ApplyDryRun, os.Stdin)
		if err != nil {
			return fmt.Errorf("dry-run: %w", err)
		}
		printFixSummary(applyResult, fixable)
		return nil
	}

	if apply {
		var originalBranch string

		if gitBranch != "" {
			var err error
			originalBranch, err = setupGitBranch(gitBranch)
			if err != nil {
				return err
			}
		}

		mode := fixer.ApplyInteractive
		if yes {
			mode = fixer.ApplyAll
		}

		applyResult, err := fixer.ApplyFixes(result.Issues, mode, os.Stdin)
		if err != nil {
			return fmt.Errorf("applying fixes: %w", err)
		}

		if len(applyResult.Applied) > 0 {
			fmt.Printf("\n  🔧 Applied %d fix(es):\n", len(applyResult.Applied))
			for _, a := range applyResult.Applied {
				fmt.Printf("     ✅ %s:%d — %s\n", a.File, a.Line, a.Title)
			}
		}

		if len(applyResult.Skipped) > 0 {
			fmt.Printf("\n  ⏭️  Skipped %d fix(es):\n", len(applyResult.Skipped))
			for _, s := range applyResult.Skipped {
				fmt.Printf("     ⚠️  %s:%d — %s (%s)\n", s.File, s.Line, s.Title, s.Reason)
			}
		}

		printFixSummary(applyResult, fixable)

		if originalBranch != "" && len(applyResult.Applied) > 0 {
			fmt.Printf("  🌿 Fixes applied on branch: %s\n", gitBranch)
			fmt.Printf("     To revert: git checkout %s\n", originalBranch)
			fmt.Printf("     To review: git diff %s...%s\n\n", originalBranch, gitBranch)
		}
	}

	return nil
}

func setupGitBranch(branchName string) (originalBranch string, err error) {
	if !git.IsCleanWorkingTree() {
		return "", fmt.Errorf("working tree has uncommitted changes — commit or stash before using --git-branch")
	}

	originalBranch, err = git.GetCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("cannot determine current branch: %w", err)
	}

	if branchName == "auto" {
		branchName = fmt.Sprintf("larasense-fix/%s", time.Now().Format("2006-01-02-150405"))
	}

	if err := git.CreateAndCheckoutBranch(branchName); err != nil {
		return "", fmt.Errorf("creating fix branch: %w", err)
	}

	log.Printf("Created and switched to branch: %s", branchName)
	return originalBranch, nil
}

func printFixSummary(result *fixer.ApplyResult, totalFixable int) {
	applied := len(result.Applied)
	skipped := len(result.Skipped)

	skippedByReason := make(map[string]int)
	for _, s := range result.Skipped {
		skippedByReason[s.Reason]++
	}

	fmt.Println()
	fmt.Println("  ══════════════════════════════════════════════════════")
	fmt.Println("  Fix Summary")
	fmt.Println("  ──────────────────────────────────────────────────────")
	fmt.Printf("    Total fixable:  %d\n", totalFixable)
	fmt.Printf("    ✅ Applied:     %d\n", applied)
	fmt.Printf("    ⏭️  Skipped:     %d\n", skipped)

	if len(skippedByReason) > 0 {
		fmt.Println("    ─────────────────────────────")
		for reason, count := range skippedByReason {
			fmt.Printf("      • %s: %d\n", reason, count)
		}
	}

	fmt.Println("  ══════════════════════════════════════════════════════")

	if applied == 0 && skipped > 0 {
		fmt.Println("\n  💡 Tip: fixes were skipped because the code didn't match AI's suggestion.")
		fmt.Println("     Try running with --no-cache to get fresh fix data.")
	}

	fmt.Println()
}
