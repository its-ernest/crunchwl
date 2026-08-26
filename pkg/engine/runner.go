// Package engine drives wordlist generation: it splits the work across CPU
// cores, runs the generator concurrently, and merges the per-core output files
// into the final wordlist.
package engine

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/its-ernest/crunchwl/pkg/generator"
	"github.com/its-ernest/crunchwl/pkg/pattern"
)

// Execute expands pat into every matching word and writes them to outputPath.
//
// Work is partitioned by restricting the first variable slot of each word to a
// disjoint slice of the charset, so the workers cover the full word space
// without overlap. The number of workers is clamped to the charset size to
// avoid idle goroutines.
func Execute(pat *pattern.CompiledPattern, charset []byte, minW, maxW int, outputPath string, numWorkers int) error {
	// Clamp active workers to the length of the charset to prevent idle goroutines
	if numWorkers > len(charset) {
		numWorkers = len(charset)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	tempDir, err := os.MkdirTemp("", "crunchwl_parts_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	var wg sync.WaitGroup
	partFiles := make([]string, numWorkers)
	chunkSize := (len(charset) + numWorkers - 1) / numWorkers

	for i := 0; i < numWorkers; i++ {
		startIdx := i * chunkSize
		endIdx := startIdx + chunkSize
		if startIdx >= len(charset) {
			break
		}
		if endIdx > len(charset) {
			endIdx = len(charset)
		}

		partPath := filepath.Join(tempDir, fmt.Sprintf("part_%d.tmp", i))
		partFiles[i] = partPath
		workerSlice := charset[startIdx:endIdx]

		wg.Add(1)
		go func(workerID int, slice []byte, path string) {
			defer wg.Done()
			f, err := os.Create(path)
			if err != nil {
				return
			}
			defer f.Close()

			w := bufio.NewWriterSize(f, 4*1024*1024)
			defer w.Flush()

			cfg := generator.Config{
				Pattern:        pat,
				Charset:        charset,
				MinWildcard:    minW,
				MaxWildcard:    maxW,
				FirstCharRange: slice,
			}
			generator.Generate(cfg, w)
		}(i, workerSlice, partPath)
	}

	wg.Wait()
	return mergeFiles(partFiles, outputPath)
}

// mergeFiles concatenates the per-worker part files into the final output file
// in worker order.
func mergeFiles(partFiles []string, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	outWriter := bufio.NewWriterSize(outFile, 16*1024*1024)
	defer outWriter.Flush()

	for _, path := range partFiles {
		if path == "" {
			continue
		}
		pf, err := os.Open(path)
		if err != nil {
			continue
		}
		io.Copy(outWriter, pf)
		pf.Close()
	}
	return nil
}
