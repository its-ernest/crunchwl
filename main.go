package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/its-ernest/crunchwl/pkg/engine"
	"github.com/its-ernest/crunchwl/pkg/pattern"
)

var figletString string = `
                            _                _ 
  ___ _ __ _   _ _ __   ___| |__   __      _| |
 / __| '__| | | | '_ \ / __| '_ \  \ \ /\ / / |
| (__| |  | |_| | | | | (__| | | |  \ V  V /| |
 \___|_|   \__,_|_| |_|\___|_| |_|   \_/\_/ |_|
 Version: 1.0.4
`

// clearScreen resets the terminal using ANSI escape codes so each run starts
// from a clean view. This avoids shelling out to an external `clear` binary,
// keeping behavior consistent across platforms and when stdout is piped.
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func main() {
	patStr := flag.String("pattern", "", "SQL pattern, e.g., 'admin_%_??' or 'pass___2026'")
	chars := flag.String("chars", "abcdefghijklmnopqrstuvwxyz0123456789", "Character set")
	minW := flag.Int("min-wildcard", 1, "Min length for '%' wildcard")
	maxW := flag.Int("max-wildcard", 4, "Max length for '%' wildcard")
	output := flag.String("output", "wordlist.txt", "Output target path")
	cores := flag.Int("cores", 0, "Number of CPU cores to use (0 = use all available)")
	flag.Parse()

	if *patStr == "" {
		fmt.Print(figletString)
		fmt.Println("[-] Error: -pattern flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	// Core allocation logic
	maxSystemCores := runtime.NumCPU()
	selectedCores := *cores

	if selectedCores <= 0 || selectedCores > maxSystemCores {
		selectedCores = maxSystemCores
	}

	pat, err := pattern.Parse(*patStr)
	if err != nil {
		fmt.Printf("[-] Pattern parse error: %v\n", err)
		os.Exit(1)
	}

	// The engine partitions work across at most len(charset) workers, so the
	// effective parallelism is the smaller of the requested cores and the set.
	effectiveCores := selectedCores
	if effectiveCores > len(*chars) {
		effectiveCores = len(*chars)
	}
	if effectiveCores < 1 {
		effectiveCores = 1
	}

	//clearScreen() // Intentionally commented out
	fmt.Print(figletString)

	count := engine.Count(pat, []byte(*chars), *minW, *maxW)

	fmt.Printf("[+] Pattern: %s\n", *patStr)
	fmt.Printf("[+] Total words: %s\n", count.String())
	fmt.Printf("[+] Utilizing %d of %d available CPU Cores\n", effectiveCores, maxSystemCores)

	start := time.Now()
	if err := engine.Execute(pat, []byte(*chars), *minW, *maxW, *output, selectedCores); err != nil {
		fmt.Printf("[-] Execution error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[+] Finished. Output saved to %s in %v\n", *output, time.Since(start))
}
