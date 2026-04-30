package ui

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var isTTY = isTerminal()

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func ProgressBar(current, total int, label string) {
	if !isTTY || total == 0 {
		return
	}

	pct := current * 100 / total
	barWidth := 20
	filled := barWidth * current / total
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	if len(label) > 40 {
		label = label[:37] + "..."
	}

	fmt.Fprintf(os.Stderr, "\r  [%s] %d/%d (%d%%) — %s", bar, current, total, pct, label)

	if current == total {
		fmt.Fprintln(os.Stderr)
	}
}

func PrintHeader(title string, fields map[string]string, order []string) {
	maxKey := 0
	for _, k := range order {
		if len(k) > maxKey {
			maxKey = len(k)
		}
	}

	fmt.Println()
	fmt.Println("  ┌──────────────────────────────────────────────────────")
	fmt.Printf("  │ %s\n", title)
	fmt.Println("  │")
	for _, k := range order {
		v := fields[k]
		fmt.Printf("  │ %-*s  %s\n", maxKey+1, k+":", v)
	}
	fmt.Println("  └──────────────────────────────────────────────────────")
	fmt.Println()
}

func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

func FormatSkipReason(file string, line int, title, reason string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "     ⚠️  SKIPPED: %s:%d\n", file, line)
	fmt.Fprintf(&sb, "        Issue: %s\n", title)

	short, tip := splitReasonAndTip(reason)
	fmt.Fprintf(&sb, "        Reason: %s\n", short)
	if tip != "" {
		fmt.Fprintf(&sb, "        💡 %s\n", tip)
	}

	return sb.String()
}

func splitReasonAndTip(reason string) (short, tip string) {
	if idx := strings.Index(reason, " (re-run"); idx >= 0 {
		return reason[:idx], "Re-run with --fresh to get updated fix data"
	}
	if idx := strings.Index(reason, " — file may"); idx >= 0 {
		return reason[:idx], "File may have changed since analysis — re-run with --fresh"
	}
	if strings.Contains(reason, "shifted by a previously applied fix") {
		return "Code shifted by a previously applied fix", "Re-run with --fresh to get updated fix data"
	}
	return reason, ""
}
