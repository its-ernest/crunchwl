# crunchwl

A modern, optimized reimplementation of the classic `crunch` wordlist generator,
written in Go. It expands SQL-style patterns containing wildcards into every
matching word and writes them to a file, parallelized across CPU cores.

## Build

```sh
go build -o crunchwl .
```

## Usage

```sh
./crunchwl -pattern "admin_%_??" -chars "abcdefghijklmnopqrstuvwxyz0123456789" -min-wildcard 1 -max-wildcard 4 -cores 0 -output wordlist.txt
```

## Example output

```sh
[zsh@arch] (.../my/projects/its-ernest/crunchwl) % ./crunchwl -pattern "%" -chars "abedipr.8" -min-wildcard 8 -max-wildcard 10 -output output.txt -cores 5
                            _                _ 
  ___ _ __ _   _ _ __   ___| |__   __      _| |
 / __| '__| | | | '_ \ / __| '_ \  \ \ /\ / / |
| (__| |  | |_| | | | | (__| | | |  \ V  V /| |
 \___|_|   \__,_|_| |_|\___|_| |_|   \_/\_/ |_|
 Version: 1.0.0
[+] Pattern: %
[+] Total words: 3917251611
[+] Timeline: Approximately 24 seconds
[+] Utilizing 5 of 8 available CPU Cores
[+] Finished. Output saved to output.txt in 17m3.539446551s
```

### Flags

| Flag             | Default                                  | Description                                              |
| ---------------- | ---------------------------------------- | -------------------------------------------------------- |
| `-pattern`       | _(required)_                             | Pattern to expand, e.g. `admin_%_??` or `pass___2026`.   |
| `-chars`         | `a-z0-9`                                 | Character set used to fill wildcards.                    |
| `-min-wildcard`  | `1`                                      | Minimum length of a `%` (multi) wildcard.                |
| `-max-wildcard`  | `4`                                      | Maximum length of a `%` (multi) wildcard.                |
| `-output`        | `wordlist.txt`                           | Destination file for the generated wordlist.             |
| `-cores`         | `0` (use all available cores)            | Number of CPU cores to use; clamped to the charset size. |

## Pattern syntax

Patterns are sequences of three token kinds:

- **Literal text** — emitted verbatim into every word (e.g. `admin`).
- **`_` (single)** — exactly one character from the charset.
- **`%` (multi)** — a run of characters whose length is chosen independently from
  `[min-wildcard, max-wildcard]` for each `%` in the pattern.

Example: `admin_%_??` expands to words like `admin`, a 1–4 character wildcard
filler, a single charset character, then the literal `??`.
