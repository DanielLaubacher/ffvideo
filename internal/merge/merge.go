package merge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DanielLaubacher/ffvideo/internal/ffmpeg"
)

// Config holds all parameters for the merge operation.
type Config struct {
	Output   string
	ListFile string
	Files    []string
}

// Validate checks that required fields are set and input files exist.
func (c *Config) Validate() error {
	if c.Output == "" {
		return fmt.Errorf("output file required")
	}
	if c.ListFile == "" && len(c.Files) < 2 {
		return fmt.Errorf("at least 2 input files required")
	}
	if c.ListFile != "" {
		if _, err := os.Stat(c.ListFile); err != nil {
			return fmt.Errorf("list file not found: %s", c.ListFile)
		}
	}
	for _, f := range c.Files {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("file not found: %s", f)
		}
	}
	return nil
}

// Run executes the merge operation.
func Run(ctx context.Context, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	// For positional file args, check if inputs need re-encoding
	if cfg.ListFile == "" && !inputsMatch(cfg.Files) {
		fmt.Fprintf(os.Stderr, "Inputs have different codecs/resolutions, re-encoding %d files...\n", len(cfg.Files))
		absFiles := make([]string, len(cfg.Files))
		for i, f := range cfg.Files {
			abs, err := filepath.Abs(f)
			if err != nil {
				return fmt.Errorf("resolve path %s: %w", f, err)
			}
			absFiles[i] = abs
		}
		args := ffmpeg.ConcatFilter(absFiles, cfg.Output)
		if err := ffmpeg.RunSilent(ctx, args); err != nil {
			return fmt.Errorf("merge: %w", err)
		}
		reportSize(cfg.Output)
		return nil
	}

	tmpdir, err := os.MkdirTemp("", "ffv-merge-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	concatFile := filepath.Join(tmpdir, "concat.txt")

	if cfg.ListFile != "" {
		if err := buildConcatFromListFile(cfg.ListFile, concatFile); err != nil {
			return err
		}
	} else {
		if err := buildConcatFromFiles(cfg.Files, concatFile); err != nil {
			return err
		}
	}

	// Count entries for status message
	data, _ := os.ReadFile(concatFile)
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "file ") {
			count++
		}
	}
	if count == 0 {
		return fmt.Errorf("no valid input files")
	}

	fmt.Fprintf(os.Stderr, "Merging %d files...\n", count)

	args := ffmpeg.Concat(ffmpeg.ConcatArgs{ListFile: concatFile, Output: cfg.Output})
	if err := ffmpeg.RunSilent(ctx, args); err != nil {
		return fmt.Errorf("merge: %w", err)
	}

	reportSize(cfg.Output)
	return nil
}

// inputsMatch returns true if all files share the same video codec and resolution.
func inputsMatch(files []string) bool {
	if len(files) < 2 {
		return true
	}
	refCodec, err := ffmpeg.VideoCodec(files[0])
	if err != nil {
		return true // assume match if probe fails, let concat try
	}
	refW, refH, err := ffmpeg.Resolution(files[0])
	if err != nil {
		return true
	}
	for _, f := range files[1:] {
		codec, err := ffmpeg.VideoCodec(f)
		if err != nil || codec != refCodec {
			return false
		}
		w, h, err := ffmpeg.Resolution(f)
		if err != nil || w != refW || h != refH {
			return false
		}
	}
	return true
}

func reportSize(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	size := float64(info.Size()) / (1024 * 1024)
	fmt.Fprintf(os.Stderr, "Done: %s (%.1f MB)\n", path, size)
}

func buildConcatFromFiles(files []string, concatFile string) error {
	var lines []string
	for _, f := range files {
		abs, err := filepath.Abs(f)
		if err != nil {
			return fmt.Errorf("resolve path %s: %w", f, err)
		}
		lines = append(lines, fmt.Sprintf("file '%s'", abs))
	}
	return os.WriteFile(concatFile, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func buildConcatFromListFile(listFile, concatFile string) error {
	data, err := os.ReadFile(listFile)
	if err != nil {
		return fmt.Errorf("read list file: %w", err)
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, err := os.Stat(line); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: '%s' not found, skipping.\n", line)
			continue
		}
		abs, err := filepath.Abs(line)
		if err != nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("file '%s'", abs))
	}

	if len(lines) == 0 {
		return fmt.Errorf("no valid files found in %s", listFile)
	}
	return os.WriteFile(concatFile, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
