package edit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/DanielLaubacher/ffvideo/internal/ffmpeg"
)

// Config holds all parameters for the edit pipeline.
type Config struct {
	Input       string
	Output      string
	TrimStart   string
	TrimEnd     string
	Normalize   bool
	Denoise     bool
	AudioFile   string
	AudioVolume string
	Text        string
	TextColor   string
	TextSize    int
	TextBold    bool
}

// Validate checks that required fields are set and files exist.
func (c *Config) Validate() error {
	if c.Input == "" {
		return fmt.Errorf("input file required")
	}
	if c.Output == "" {
		return fmt.Errorf("output file required")
	}
	if _, err := os.Stat(c.Input); err != nil {
		return fmt.Errorf("input file not found: %s", c.Input)
	}
	if c.AudioFile != "" {
		if _, err := os.Stat(c.AudioFile); err != nil {
			return fmt.Errorf("audio file not found: %s", c.AudioFile)
		}
	}
	return nil
}

// Run executes the edit pipeline: trim → normalize → denoise → audio → text.
func Run(ctx context.Context, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Fast path: trim-only uses stream copy (no re-encode, near-instant)
	trimOnly := (cfg.TrimStart != "" || cfg.TrimEnd != "") &&
		!cfg.Normalize && !cfg.Denoise && cfg.AudioFile == "" && cfg.Text == ""
	if trimOnly {
		fmt.Fprintln(os.Stderr, "Trimming (stream copy)...")
		args := ffmpeg.TrimCopy(ffmpeg.TrimArgs{
			Input:  cfg.Input,
			Output: cfg.Output,
			Start:  cfg.TrimStart,
			End:    cfg.TrimEnd,
		})
		if err := ffmpeg.RunSilent(ctx, args); err != nil {
			return fmt.Errorf("trim: %w", err)
		}
		reportSize(cfg.Output)
		return nil
	}

	tmpdir, err := os.MkdirTemp("", "ffv-edit-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	current := cfg.Input
	step := 0

	// Trim
	if cfg.TrimStart != "" || cfg.TrimEnd != "" {
		step++
		next := filepath.Join(tmpdir, fmt.Sprintf("%d_trimmed.mp4", step))
		msg := "Trimming"
		if cfg.TrimStart != "" {
			msg += " from " + cfg.TrimStart
		}
		if cfg.TrimEnd != "" {
			msg += " to " + cfg.TrimEnd
		}
		fmt.Fprintln(os.Stderr, msg+"...")

		args := ffmpeg.Trim(ffmpeg.TrimArgs{
			Input:  current,
			Output: next,
			Start:  cfg.TrimStart,
			End:    cfg.TrimEnd,
		})
		if err := ffmpeg.RunSilent(ctx, args); err != nil {
			return fmt.Errorf("trim: %w", err)
		}
		current = next
	}

	// Normalize audio
	if cfg.Normalize {
		step++
		next := filepath.Join(tmpdir, fmt.Sprintf("%d_normalized.mp4", step))
		fmt.Fprintln(os.Stderr, "Normalizing audio (EBU R128)...")

		args := ffmpeg.Normalize(ffmpeg.NormalizeArgs{Input: current, Output: next})
		if err := ffmpeg.RunSilent(ctx, args); err != nil {
			return fmt.Errorf("normalize: %w", err)
		}
		current = next
	}

	// Denoise
	if cfg.Denoise {
		step++
		next := filepath.Join(tmpdir, fmt.Sprintf("%d_denoised.mp4", step))
		fmt.Fprintln(os.Stderr, "Reducing noise...")

		args := ffmpeg.Denoise(ffmpeg.DenoiseArgs{Input: current, Output: next})
		if err := ffmpeg.RunSilent(ctx, args); err != nil {
			return fmt.Errorf("denoise: %w", err)
		}
		current = next
	}

	// Background audio
	if cfg.AudioFile != "" {
		step++
		next := filepath.Join(tmpdir, fmt.Sprintf("%d_audio.mp4", step))

		duration, err := ffmpeg.Duration(current)
		if err != nil {
			return fmt.Errorf("probe duration: %w", err)
		}
		hasAudio, err := ffmpeg.HasAudioStream(current)
		if err != nil {
			return fmt.Errorf("probe audio: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Adding background audio (volume: %s)...\n", cfg.AudioVolume)

		args := ffmpeg.MixAudio(ffmpeg.MixAudioArgs{
			VideoInput: current,
			AudioInput: cfg.AudioFile,
			Output:     next,
			Volume:     cfg.AudioVolume,
			Duration:   duration,
			HasAudio:   hasAudio,
		})
		if err := ffmpeg.RunSilent(ctx, args); err != nil {
			return fmt.Errorf("mix audio: %w", err)
		}
		current = next
	}

	// Banner text
	if cfg.Text != "" {
		step++
		next := filepath.Join(tmpdir, fmt.Sprintf("%d_text.mp4", step))

		// Write text to file to avoid ffmpeg filter escaping issues
		textFile := filepath.Join(tmpdir, "banner.txt")
		if err := os.WriteFile(textFile, []byte(cfg.Text), 0644); err != nil {
			return fmt.Errorf("write text file: %w", err)
		}

		// Find font via fontconfig
		fontFile := findFont(cfg.TextBold)

		fmt.Fprintf(os.Stderr, "Adding text: '%s'...\n", cfg.Text)

		args := ffmpeg.DrawText(ffmpeg.DrawTextArgs{
			Input:    current,
			Output:   next,
			TextFile: textFile,
			FontFile: fontFile,
			Color:    cfg.TextColor,
			Size:     cfg.TextSize,
		})
		if err := ffmpeg.RunSilent(ctx, args); err != nil {
			return fmt.Errorf("draw text: %w", err)
		}
		current = next
	}

	// Copy final result to output
	data, err := os.ReadFile(current)
	if err != nil {
		return fmt.Errorf("read result: %w", err)
	}
	if err := os.WriteFile(cfg.Output, data, 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	reportSize(cfg.Output)
	return nil
}

func reportSize(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	size := float64(info.Size()) / (1024 * 1024)
	fmt.Fprintf(os.Stderr, "Done: %s (%.1f MB)\n", path, size)
}

// findFont uses fc-match to locate a suitable font file.
func findFont(bold bool) string {
	pattern := "sans"
	if bold {
		pattern = "sans:bold"
	}
	out, err := exec.Command("fc-match", "--format=%{file}", pattern).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
