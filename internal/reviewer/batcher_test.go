package reviewer

import (
	"strings"
	"testing"

	"github.com/Mattel-Limbo/larasense-limbo/internal/context"
)

func TestBatchFiles_SingleBatch(t *testing.T) {
	files := []context.FileContext{
		{Path: "app/Models/User.php", DiffText: "small content"},
		{Path: "app/Models/Post.php", DiffText: "small content"},
	}

	batches := batchFiles(files)

	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	if len(batches[0]) != 2 {
		t.Errorf("expected 2 files in batch, got %d", len(batches[0]))
	}
}

func TestBatchFiles_MultipleBatches(t *testing.T) {
	largeContent := strings.Repeat("x", 50_000)
	files := []context.FileContext{
		{Path: "file1.php", DiffText: largeContent},
		{Path: "file2.php", DiffText: largeContent},
		{Path: "file3.php", DiffText: largeContent},
	}

	batches := batchFiles(files)

	if len(batches) < 2 {
		t.Fatalf("expected at least 2 batches for large files, got %d", len(batches))
	}

	totalFiles := 0
	for _, b := range batches {
		totalFiles += len(b)
	}
	if totalFiles != 3 {
		t.Errorf("expected 3 total files across batches, got %d", totalFiles)
	}
}

func TestBatchFiles_OversizedFile(t *testing.T) {
	hugeContent := strings.Repeat("x", maxBatchBytes+1000)
	files := []context.FileContext{
		{Path: "small.php", DiffText: "small"},
		{Path: "huge.php", DiffText: hugeContent},
		{Path: "small2.php", DiffText: "small"},
	}

	batches := batchFiles(files)

	if len(batches) < 2 {
		t.Fatalf("expected at least 2 batches (huge file gets own batch), got %d", len(batches))
	}

	foundHuge := false
	for _, b := range batches {
		for _, f := range b {
			if f.Path == "huge.php" {
				if len(b) != 1 {
					t.Error("huge file should be alone in its batch")
				}
				foundHuge = true
			}
		}
	}
	if !foundHuge {
		t.Error("huge file should be present in batches")
	}
}

func TestBatchFiles_Empty(t *testing.T) {
	batches := batchFiles(nil)

	if len(batches) != 0 {
		t.Errorf("expected 0 batches for nil input, got %d", len(batches))
	}
}
