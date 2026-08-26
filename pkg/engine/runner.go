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
// without overlap. The number of workers is clamped to the charset size to avoid
// idle goroutines.
//
// Part files are staged in the same directory as outputPath (or $TMPDIR when
// set) so they live on the same filesystem as the final file. Any write or
// merge failure is returned as an error rather than silently producing a
// truncated wordlist.
func Execute(pat *pattern.CompiledPattern, charset []byte, minW, maxW int, outputPath string, numWorkers int) error {
	// Clamp active workers to the length of the charset to prevent idle goroutines
	if numWorkers > len(charset) {
		numWorkers = len(charset)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	// Stage temp parts next to the output (or in $TMPDIR) to avoid filling a
	// small/tmpfs with huge intermediate files.
	tempBase := filepath.Dir(outputPath)
	if t := os.Getenv("TMPDIR"); t != "" {
		tempBase = t
	}
	tempDir, err := os.MkdirTemp(tempBase, "crunchwl_parts_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	var wg sync.WaitGroup
	errCh := make(chan error, numWorkers)
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
		go func(slice []byte, path string) {
			defer wg.Done()
			f, err := os.Create(path)
			if err != nil {
				errCh <- err
				return
			}
			defer f.Close()

			w := bufio.NewWriterSize(f, 4*1024*1024)
			cfg := generator.Config{
				Pattern:        pat,
				Charset:        charset,
				MinWildcard:    minW,
				MaxWildcard:    maxW,
				FirstCharRange: slice,
			}
			if err := generator.Generate(cfg, w); err != nil {
				errCh <- err
				return
			}
			if err := w.Flush(); err != nil {
				errCh <- err
				return
			}
		}(workerSlice, partPath)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return mergeFiles(partFiles, outputPath)
}

// mergeFiles concatenates the per-worker part files into the final output file
// in worker order. Any read or write error is returned.
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
			return err
		}
		if _, err := io.Copy(outWriter, pf); err != nil {
			pf.Close()
			return err
		}
		pf.Close()
	}
	return outWriter.Flush()
}
