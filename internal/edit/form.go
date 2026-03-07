package edit

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"

	"github.com/DanielLaubacher/ffvideo/internal/ui"
)

// RunForm launches an interactive form to collect edit pipeline config.
func RunForm(cfg *Config) error {
	// Step 1: Input/output files
	fileForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Input file").
				Placeholder("input.mp4").
				Value(&cfg.Input).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("input file is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Output file").
				Placeholder("output.mp4").
				Value(&cfg.Output).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("output file is required")
					}
					return nil
				}),
		),
	).WithTheme(ui.Theme())

	if err := fileForm.Run(); err != nil {
		return err
	}

	// Step 2: Select operations
	var operations []string
	opForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Operations to apply (space to select, enter to confirm)").
				Options(
					huh.NewOption[string]("Trim (cut start/end)", "trim"),
					huh.NewOption[string]("Normalize audio (EBU R128)", "normalize"),
					huh.NewOption[string]("Reduce noise", "denoise"),
					huh.NewOption[string]("Add background audio", "audio"),
					huh.NewOption[string]("Add banner text", "text"),
				).
				Value(&operations),
		),
	).WithTheme(ui.Theme())

	if err := opForm.Run(); err != nil {
		return err
	}

	// Step 3: Collect parameters for each selected operation
	for _, op := range operations {
		switch op {
		case "trim":
			if err := runTrimForm(cfg); err != nil {
				return err
			}
		case "normalize":
			cfg.Normalize = true
		case "denoise":
			cfg.Denoise = true
		case "audio":
			if err := runAudioForm(cfg); err != nil {
				return err
			}
		case "text":
			if err := runTextForm(cfg); err != nil {
				return err
			}
		}
	}

	return nil
}

func runTrimForm(cfg *Config) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Trim start (leave blank to keep from beginning)").
				Placeholder("00:00:05").
				Value(&cfg.TrimStart),

			huh.NewInput().
				Title("Trim end (leave blank to keep to end)").
				Placeholder("00:01:30").
				Value(&cfg.TrimEnd),
		),
	).WithTheme(ui.Theme()).Run()
}

func runAudioForm(cfg *Config) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Background audio file").
				Placeholder("music.mp3").
				Value(&cfg.AudioFile).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("audio file is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Audio volume (0-1)").
				Placeholder("0.2").
				Value(&cfg.AudioVolume),
		),
	).WithTheme(ui.Theme()).Run()
}

func runTextForm(cfg *Config) error {
	textSizeStr := strconv.Itoa(cfg.TextSize)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Banner text").
				Placeholder("My Video").
				Value(&cfg.Text).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("text is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Text color").
				Placeholder("white").
				Value(&cfg.TextColor),

			huh.NewInput().
				Title("Text size").
				Placeholder("24").
				Value(&textSizeStr).
				Validate(func(s string) error {
					n, err := strconv.Atoi(s)
					if err != nil || n <= 0 {
						return fmt.Errorf("must be a positive number")
					}
					return nil
				}),

			huh.NewConfirm().
				Title("Bold font?").
				Value(&cfg.TextBold),
		),
	).WithTheme(ui.Theme())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.TextSize, _ = strconv.Atoi(textSizeStr)
	return nil
}
