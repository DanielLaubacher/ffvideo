package convert

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DanielLaubacher/ffvideo/internal/ffmpeg"
)

// Config holds parameters for format conversion.
type Config struct {
	Input  string
	Output string
	Width  int // 0 = keep original
	FPS    int // 0 = use sane default per format
}

// Validate checks required fields and file existence.
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
	ext := OutputExt(c.Output)
	if ext == "" {
		return fmt.Errorf("could not determine output format from extension: %s", c.Output)
	}
	return nil
}

// OutputExt returns the lowercase extension without the dot.
func OutputExt(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}

// SupportedFormats returns the list of output formats convert handles.
func SupportedFormats() []string {
	return []string{"gif", "webp", "webm", "mp4"}
}

// Run converts the input to the output format inferred from the extension.
func Run(ctx context.Context, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	ext := OutputExt(cfg.Output)

	switch ext {
	case "gif":
		return convertGIF(ctx, cfg)
	case "webp":
		return convertWebP(ctx, cfg)
	case "webm":
		return convertWebM(ctx, cfg)
	case "mp4":
		return convertMP4(ctx, cfg)
	default:
		return fmt.Errorf("unsupported output format: .%s (supported: gif, webp, webm, mp4)", ext)
	}
}

func convertGIF(ctx context.Context, cfg *Config) error {
	fps := cfg.FPS
	if fps == 0 {
		fps = 15 // GIF default — keeps file size reasonable
	}
	width := cfg.Width
	if width == 0 {
		width = 640 // GIF default — full resolution GIFs are enormous
	}

	fmt.Fprintf(os.Stderr, "Converting to GIF (%dx, %dfps)...\n", width, fps)
	args := ffmpeg.GIF(cfg.Input, cfg.Output, width, fps)
	if err := ffmpeg.RunSilent(ctx, args); err != nil {
		return fmt.Errorf("gif conversion: %w", err)
	}

	reportSize(cfg.Output)
	return nil
}

func convertWebP(ctx context.Context, cfg *Config) error {
	fps := cfg.FPS
	if fps == 0 {
		fps = 15
	}
	width := cfg.Width
	if width == 0 {
		width = 640
	}

	fmt.Fprintf(os.Stderr, "Converting to WebP (%dx, %dfps)...\n", width, fps)
	args := ffmpeg.ConvertToWebP(ffmpeg.ConvertArgs{
		Input:  cfg.Input,
		Output: cfg.Output,
		Width:  width,
		FPS:    fps,
	})
	if err := ffmpeg.RunSilent(ctx, args); err != nil {
		return fmt.Errorf("webp conversion: %w", err)
	}

	reportSize(cfg.Output)
	return nil
}

func convertWebM(ctx context.Context, cfg *Config) error {
	fmt.Fprintln(os.Stderr, "Converting to WebM (VP9)...")
	args := ffmpeg.ConvertToWebM(ffmpeg.ConvertArgs{
		Input:  cfg.Input,
		Output: cfg.Output,
		Width:  cfg.Width,
		FPS:    cfg.FPS,
	})
	if err := ffmpeg.RunSilent(ctx, args); err != nil {
		return fmt.Errorf("webm conversion: %w", err)
	}

	reportSize(cfg.Output)
	return nil
}

func convertMP4(ctx context.Context, cfg *Config) error {
	fmt.Fprintln(os.Stderr, "Re-encoding to MP4 (h264)...")
	args := []string{
		"ffmpeg", "-hide_banner", "-y", "-i", cfg.Input,
	}

	vf := ""
	if cfg.FPS > 0 {
		vf = fmt.Sprintf("fps=%d", cfg.FPS)
	}
	if cfg.Width > 0 {
		if vf != "" {
			vf += ","
		}
		vf += fmt.Sprintf("scale=%d:-2:flags=lanczos", cfg.Width)
	}
	if vf != "" {
		args = append(args, "-vf", vf)
	}

	args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-movflags", "+faststart", "-c:a", "aac", cfg.Output)
	if err := ffmpeg.RunSilent(ctx, args); err != nil {
		return fmt.Errorf("mp4 conversion: %w", err)
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
