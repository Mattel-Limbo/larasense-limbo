package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	// Set known values
	Version = "1.2.3"
	Commit = "abc1234"
	BuildDate = "2026-01-01"
	defer func() {
		Version = "dev"
		Commit = "none"
		BuildDate = "unknown"
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "1.2.3") {
		t.Errorf("output should contain version '1.2.3', got: %s", out)
	}
	if !strings.Contains(out, "abc1234") {
		t.Errorf("output should contain commit 'abc1234', got: %s", out)
	}
	if !strings.Contains(out, "2026-01-01") {
		t.Errorf("output should contain build date '2026-01-01', got: %s", out)
	}
}

func TestInitCommand_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("init command error: %v", err)
	}

	// File should exist
	content, err := os.ReadFile(filepath.Join(dir, ".larasense-limbo.yml"))
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Should contain key sections
	s := string(content)
	if !strings.Contains(s, "provider:") {
		t.Error("config should contain 'provider:' section")
	}
	if !strings.Contains(s, "review:") {
		t.Error("config should contain 'review:' section")
	}
	if !strings.Contains(s, "filters:") {
		t.Error("config should contain 'filters:' section")
	}
	if !strings.Contains(s, "endpoint:") {
		t.Error("config should contain 'endpoint:' field")
	}
}

func TestInitCommand_RefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create existing file
	os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte("existing"), 0644)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("init should fail when file exists without --force")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %s", err.Error())
	}

	// Original file should be untouched
	content, _ := os.ReadFile(filepath.Join(dir, ".larasense-limbo.yml"))
	if string(content) != "existing" {
		t.Error("existing file should not be overwritten")
	}
}

func TestInitCommand_ForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create existing file
	os.WriteFile(filepath.Join(dir, ".larasense-limbo.yml"), []byte("old-content"), 0644)

	// Reset flagForce for this test
	flagForce = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init", "--force"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("init --force should succeed, got error: %v", err)
	}

	// File should be overwritten with template
	content, _ := os.ReadFile(filepath.Join(dir, ".larasense-limbo.yml"))
	if string(content) == "old-content" {
		t.Error("file should be overwritten with --force")
	}
	if !strings.Contains(string(content), "provider:") {
		t.Error("overwritten file should contain config template")
	}
}

func TestAnalyzeCommand_Flags(t *testing.T) {
	// Verify flags are registered
	f := analyzeCmd.Flags()

	baseFlag := f.Lookup("base")
	if baseFlag == nil {
		t.Fatal("--base flag not registered")
	}
	if baseFlag.DefValue != "origin/main" {
		t.Errorf("--base default = %q, want 'origin/main'", baseFlag.DefValue)
	}

	headFlag := f.Lookup("head")
	if headFlag == nil {
		t.Fatal("--head flag not registered")
	}
	if headFlag.DefValue != "HEAD" {
		t.Errorf("--head default = %q, want 'HEAD'", headFlag.DefValue)
	}

	jsonFlag := f.Lookup("json")
	if jsonFlag == nil {
		t.Fatal("--json flag not registered")
	}
	if jsonFlag.DefValue != "false" {
		t.Errorf("--json default = %q, want 'false'", jsonFlag.DefValue)
	}

	verboseFlag := f.Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("--verbose flag not registered")
	}
	if verboseFlag.DefValue != "false" {
		t.Errorf("--verbose default = %q, want 'false'", verboseFlag.DefValue)
	}
}

func TestRootCommand_HasSubcommands(t *testing.T) {
	commands := rootCmd.Commands()

	expected := map[string]bool{
		"analyze": false,
		"init":    false,
		"version": false,
	}

	for _, cmd := range commands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("subcommand '%s' not registered on root command", name)
		}
	}
}
