package convert

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"

	"github.com/DanielLaubacher/ffvideo/internal/ui"
)

// RunForm launches an interactive form to collect convert config.
func RunForm(cfg *Config) error {
	widthStr := ""
	if cfg.Width > 0 {
		widthStr = strconv.Itoa(cfg.Width)
	}
	fpsStr := ""
	if cfg.FPS > 0 {
		fpsStr = strconv.Itoa(cfg.FPS)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Input file").
				Placeholder("recording.mp4").
				Value(&cfg.Input).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("input file is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Output file (format inferred from extension)").
				Placeholder("output.gif").
				Value(&cfg.Output).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("output file is required")
					}
					ext := OutputExt(s)
					switch ext {
					case "gif", "webp", "webm", "mp4":
						return nil
					default:
						return fmt.Errorf("supported: .gif, .webp, .webm, .mp4")
					}
				}),

			huh.NewInput().
				Title("Width (blank = format default: 640 for gif/webp, original for webm/mp4)").
				Placeholder("640").
				Value(&widthStr).
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					n, err := strconv.Atoi(s)
					if err != nil || n <= 0 {
						return fmt.Errorf("must be a positive number")
					}
					return nil
				}),

			huh.NewInput().
				Title("FPS (blank = format default: 15 for gif/webp, original for webm/mp4)").
				Placeholder("15").
				Value(&fpsStr).
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
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

	if widthStr != "" {
		cfg.Width, _ = strconv.Atoi(widthStr)
	}
	if fpsStr != "" {
		cfg.FPS, _ = strconv.Atoi(fpsStr)
	}
	return nil
}
