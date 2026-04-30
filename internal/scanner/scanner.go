package scanner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
	"github.com/Mattel-Limbo/larasense-limbo/internal/context"
)

type Scanner struct {
	cfg        *config.Config
	scanDir    string
	projectDir string
}

func New(cfg *config.Config, scanDir string) *Scanner {
	projectDir, err := os.Getwd()
	if err != nil {
		projectDir = scanDir
	}
	return &Scanner{cfg: cfg, scanDir: scanDir, projectDir: projectDir}
}

func (s *Scanner) Scan() ([]context.FileContext, error) {
	var files []context.FileContext

	absScandDir, err := resolveAbsPath(s.scanDir)
	if err != nil {
		return nil, fmt.Errorf("resolving scan path: %w", err)
	}

	absProjectDir, err := resolveAbsPath(s.projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving project path: %w", err)
	}

	err = filepath.WalkDir(absScandDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return s.skipDir(d.Name())
		}

		// Path relative to project root (for filter matching: app/**, routes/**, etc.)
		relPath, err := filepath.Rel(absProjectDir, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		if !context.ShouldIncludeFile(relPath, s.cfg) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		files = append(files, context.FileContext{
			Path:     relPath,
			Type:     context.ClassifyFile(relPath),
			Hint:     context.GenerateHint(relPath),
			DiffText: string(content),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("scanning directory %s: %w", s.scanDir, err)
	}

	return files, nil
}

func (s *Scanner) ScanFiles(filePaths []string) ([]context.FileContext, error) {
	absProjectDir, err := resolveAbsPath(s.projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving project path: %w", err)
	}

	var files []context.FileContext

	for _, filePath := range filePaths {
		absPath, err := resolveAbsPath(filePath)
		if err != nil {
			return nil, fmt.Errorf("resolving file path %s: %w", filePath, err)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%s is a directory, use --path instead", filePath)
		}

		relPath, err := filepath.Rel(absProjectDir, absPath)
		if err != nil {
			relPath = filepath.ToSlash(filePath)
		}
		relPath = filepath.ToSlash(relPath)

		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", filePath, err)
		}

		files = append(files, context.FileContext{
			Path:     relPath,
			Type:     context.ClassifyFile(relPath),
			Hint:     context.GenerateHint(relPath),
			DiffText: string(content),
		})
	}

	return files, nil
}

func (s *Scanner) skipDir(name string) error {
	skipDirs := []string{".git", "vendor", "node_modules", ".idea", ".vscode", "storage"}
	for _, skip := range skipDirs {
		if name == skip {
			return filepath.SkipDir
		}
	}
	return nil
}

func resolveAbsPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil
	}
	return resolved, nil
}
