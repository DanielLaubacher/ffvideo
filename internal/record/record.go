package record

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DanielLaubacher/ffvideo/internal/convert"
	"github.com/DanielLaubacher/ffvideo/internal/detect"
	"github.com/DanielLaubacher/ffvideo/internal/ffmpeg"
)

// Config holds all parameters for a recording session.
type Config struct {
	Output    string
	Desktop   bool
	Silent    bool
	Framerate int
	AudioDev  string // raw device string override, e.g. "-f pulse -i default"
	Preset    string // x264 preset (ultrafast, fast, medium, etc.)
	CRF       int    // quality (0=lossless, 18=high, 23=default, 51=worst)

	// Populated by Resolve()
	width  int
	height int
	x, y   int
	audio  *ffmpeg.AudioInput
}

// Resolve fills in computed fields: audio detection and screen geometry.
func (c *Config) Resolve() error {
	if detect.DisplayServer() == "wayland" {
		fmt.Fprintln(os.Stderr, "Warning: ffmpeg x11grab does not support native Wayland capture.")
		fmt.Fprintln(os.Stderr, "Recording may work via XWayland, or use wf-recorder for native Wayland.")
	}

	// Audio
	if !c.Silent {
		if c.AudioDev != "" {
			c.audio = &ffmpeg.AudioInput{Format: "pulse", Device: c.AudioDev}
		} else {
			c.audio = detect.DetectAudio()
			if c.audio == nil {
				fmt.Fprintln(os.Stderr, "Warning: No audio device found, recording without audio.")
			}
		}
	}

	// Geometry
	if c.Desktop {
		g, err := detect.ScreenGeometry()
		if err != nil {
			return err
		}
		c.width, c.height, c.x, c.y = g.Width, g.Height, g.X, g.Y
	} else {
		fmt.Fprintln(os.Stderr, "Select a region or window...")
		g, err := detect.SelectRegion()
		if err != nil {
			return err
		}
		c.width, c.height, c.x, c.y = g.Width, g.Height, g.X, g.Y
	}

	c.width = detect.RoundEven(c.width)
	c.height = detect.RoundEven(c.height)
	return nil
}

// Validate checks that required fields are set.
func (c *Config) Validate() error {
	if c.Output == "" {
		return fmt.Errorf("output file required")
	}
	if c.Framerate <= 0 {
		return fmt.Errorf("framerate must be positive")
	}
	return nil
}

// needsConversion returns true if the output format can't be recorded directly.
func needsConversion(output string) bool {
	ext := convert.OutputExt(output)
	switch ext {
	case "mp4", "mkv", "":
		return false
	default:
		return true
	}
}

// Run executes the recording, with auto-conversion if the output extension
// requires it (e.g., .gif, .webp, .webm).
func Run(ctx context.Context, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := cfg.Resolve(); err != nil {
		return err
	}

	// If output is .gif/.webp/.webm, record to temp mp4 first, then convert
	recordOutput := cfg.Output
	var tmpdir string
	if needsConversion(cfg.Output) {
		var err error
		tmpdir, err = os.MkdirTemp("", "ffv-record-*")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpdir)
		recordOutput = filepath.Join(tmpdir, "recording.mp4")
	}

	args := ffmpeg.Record(ffmpeg.RecordArgs{
		Width:     cfg.width,
		Height:    cfg.height,
		X:         cfg.x,
		Y:         cfg.y,
		Framerate: cfg.Framerate,
		Audio:     cfg.audio,
		Preset:    cfg.Preset,
		CRF:       cfg.CRF,
		Output:    recordOutput,
	})

	fmt.Fprintf(os.Stderr, "Recording %dx%d+%d,%d @ %dfps\n",
		cfg.width, cfg.height, cfg.x, cfg.y, cfg.Framerate)
	fmt.Fprintln(os.Stderr, "Press q or Ctrl+C to stop.")

	if err := ffmpeg.Run(ctx, args); err != nil {
		return err
	}

	// Auto-convert if needed
	if needsConversion(cfg.Output) {
		fmt.Fprintf(os.Stderr, "Converting to %s...\n", convert.OutputExt(cfg.Output))
		convCfg := &convert.Config{
			Input:  recordOutput,
			Output: cfg.Output,
		}
		if err := convert.Run(ctx, convCfg); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(os.Stderr, "Saved: %s\n", cfg.Output)
	}

	fmt.Fprintf(os.Stderr, "Play:  ffvideo play %s\n", cfg.Output)
	return nil
}
