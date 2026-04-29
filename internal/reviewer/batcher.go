package reviewer

import "github.com/Mattel-Limbo/larasense-limbo/internal/context"

// ~4 chars per token. 80KB ≈ 20K tokens of content per batch,
// leaving room for system prompt (~500 tokens) + response (~2K tokens).
const maxBatchBytes = 80_000

func batchFiles(files []context.FileContext) [][]context.FileContext {
	var batches [][]context.FileContext
	var current []context.FileContext
	currentSize := 0

	for _, f := range files {
		fileSize := len(f.DiffText) + len(f.Path) + len(f.Hint) + 64

		if fileSize > maxBatchBytes {
			if len(current) > 0 {
				batches = append(batches, current)
				current = nil
				currentSize = 0
			}
			batches = append(batches, []context.FileContext{f})
			continue
		}

		if currentSize+fileSize > maxBatchBytes {
			batches = append(batches, current)
			current = nil
			currentSize = 0
		}

		current = append(current, f)
		currentSize += fileSize
	}

	if len(current) > 0 {
		batches = append(batches, current)
	}

	return batches
}
