package record

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"

	"github.com/DanielLaubacher/ffvideo/internal/detect"
	"github.com/DanielLaubacher/ffvideo/internal/ui"
)

// RunForm launches an interactive form to collect recording config.
// Fields that already have values from CLI flags are pre-filled.
func RunForm(cfg *Config) error {
	// Detect audio devices for display
	audioChoice := "auto"
	if cfg.Silent {
		audioChoice = "silent"
	} else if cfg.AudioDev != "" {
		audioChoice = cfg.AudioDev
	}

	captureMode := "region"
	if cfg.Desktop {
		captureMode = "desktop"
	}

	framerateStr := strconv.Itoa(cfg.Framerate)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Output file").
				Placeholder("recording.mp4").
				Value(&cfg.Output).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("output file is required")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Capture mode").
				Options(
					huh.NewOption("Select region (slop/slurp/xwininfo)", "region"),
					huh.NewOption("Full desktop (primary monitor)", "desktop"),
				).
				Value(&captureMode),

			huh.NewSelect[string]().
				Title("Audio").
				Options(
					buildAudioOptions()...,
				).
				Value(&audioChoice),

			huh.NewInput().
				Title("Frame rate").
				Placeholder("30").
				Value(&framerateStr).
				Validate(func(s string) error {
					n, err := strconv.Atoi(s)
					if err != nil || n <= 0 {
						return fmt.Errorf("must be a positive number")
					}
					return nil
				}),
		),
	).WithTheme(ui.Theme())

	if err := form.Run(); err != nil {
		return err
	}

	// Apply form values back to config
	cfg.Desktop = captureMode == "desktop"

	switch audioChoice {
	case "silent":
		cfg.Silent = true
	case "auto":
		cfg.Silent = false
		cfg.AudioDev = ""
	default:
		cfg.Silent = false
		cfg.AudioDev = audioChoice
	}

	cfg.Framerate, _ = strconv.Atoi(framerateStr)
	return nil
}

func buildAudioOptions() []huh.Option[string] {
	opts := []huh.Option[string]{
		huh.NewOption[string]("Auto-detect", "auto"),
		huh.NewOption[string]("Silent (no audio)", "silent"),
	}

	// If PulseAudio is available, offer it explicitly
	dev := detect.DetectAudio()
	if dev != nil {
		label := fmt.Sprintf("%s (%s)", dev.Device, dev.Format)
		opts = append(opts, huh.NewOption[string](label, fmt.Sprintf("-f %s -i %s", dev.Format, dev.Device)))
	}

	return opts
}
