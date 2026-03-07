package configcmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/DanielLaubacher/ffvideo/internal/config"
	"github.com/DanielLaubacher/ffvideo/internal/detect"
	"github.com/DanielLaubacher/ffvideo/internal/ui"
)

// NewCommand returns the "config" cobra.Command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure default settings",
		Long: `Interactive wizard to set default preferences.
Saves to ` + config.Path() + `.

Settings are used as defaults and can always be overridden by flags.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigForm()
		},
	}
	return cmd
}

func runConfigForm() error {
	cfg := config.Load()

	framerateStr := strconv.Itoa(cfg.Record.Framerate)
	crfStr := strconv.Itoa(cfg.Record.CRF)
	textSizeStr := strconv.Itoa(cfg.Edit.TextSize)

	// Build audio device options
	audioValue := cfg.Record.AudioDevice
	if audioValue == "" {
		audioValue = "auto"
	}

	// Recording defaults
	recordForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Recording Defaults"),

			huh.NewInput().
				Title("Frame rate").
				Value(&framerateStr).
				Validate(func(s string) error {
					n, err := strconv.Atoi(s)
					if err != nil || n <= 0 {
						return fmt.Errorf("must be a positive number")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Audio device").
				Options(buildAudioOptions(audioValue)...).
				Value(&audioValue),

			huh.NewConfirm().
				Title("Default to silent (no audio)?").
				Value(&cfg.Record.Silent),

			huh.NewSelect[string]().
				Title("x264 encoding preset").
				Description("Faster = larger files, slower = better compression").
				Options(
					huh.NewOption[string]("ultrafast (best for recording)", "ultrafast"),
					huh.NewOption[string]("superfast", "superfast"),
					huh.NewOption[string]("veryfast", "veryfast"),
					huh.NewOption[string]("faster", "faster"),
					huh.NewOption[string]("fast", "fast"),
					huh.NewOption[string]("medium (balanced)", "medium"),
					huh.NewOption[string]("slow (smaller files)", "slow"),
				).
				Value(&cfg.Record.Preset),

			huh.NewInput().
				Title("CRF (quality: 0=lossless, 18=high, 23=default, 28=low)").
				Value(&crfStr).
				Validate(func(s string) error {
					n, err := strconv.Atoi(s)
					if err != nil || n < 0 || n > 51 {
						return fmt.Errorf("must be 0-51")
					}
					return nil
				}),
		),
	).WithTheme(ui.Theme())

	if err := recordForm.Run(); err != nil {
		return err
	}

	cfg.Record.Framerate, _ = strconv.Atoi(framerateStr)
	cfg.Record.CRF, _ = strconv.Atoi(crfStr)
	if audioValue == "auto" {
		cfg.Record.AudioDevice = ""
	} else {
		cfg.Record.AudioDevice = audioValue
	}

	// Edit defaults
	editForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Edit Defaults"),

			huh.NewInput().
				Title("Background audio volume (0-1)").
				Value(&cfg.Edit.AudioVolume).
				Validate(func(s string) error {
					v, err := strconv.ParseFloat(s, 64)
					if err != nil || v < 0 || v > 1 {
						return fmt.Errorf("must be between 0 and 1")
					}
					return nil
				}),

			huh.NewInput().
				Title("Default text color").
				Value(&cfg.Edit.TextColor),

			huh.NewInput().
				Title("Default text size").
				Value(&textSizeStr).
				Validate(func(s string) error {
					n, err := strconv.Atoi(s)
					if err != nil || n <= 0 {
						return fmt.Errorf("must be a positive number")
					}
					return nil
				}),
		),
	).WithTheme(ui.Theme())

	if err := editForm.Run(); err != nil {
		return err
	}

	cfg.Edit.TextSize, _ = strconv.Atoi(textSizeStr)

	// Save
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Config saved to %s\n", config.Path())
	return nil
}

func buildAudioOptions(current string) []huh.Option[string] {
	opts := []huh.Option[string]{
		huh.NewOption[string]("Auto-detect", "auto"),
	}

	dev := detect.DetectAudio()
	if dev != nil {
		label := fmt.Sprintf("%s (%s)", dev.Device, dev.Format)
		val := fmt.Sprintf("-f %s -i %s", dev.Format, dev.Device)
		opts = append(opts, huh.NewOption[string](label, val))
	}

	// If the current value isn't in the list, add it
	if current != "" && current != "auto" {
		found := false
		for _, o := range opts {
			if o.Value == current {
				found = true
				break
			}
		}
		if !found {
			opts = append(opts, huh.NewOption[string](current, current))
		}
	}

	return opts
}
