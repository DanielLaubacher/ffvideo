package merge

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/DanielLaubacher/ffvideo/internal/ui"
)

// RunForm launches an interactive form to collect merge config.
func RunForm(cfg *Config) error {
	var filesStr string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Output file").
				Placeholder("merged.mp4").
				Value(&cfg.Output).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("output file is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Input files (comma-separated)").
				Placeholder("part1.mp4, part2.mp4, part3.mp4").
				Value(&filesStr).
				Validate(func(s string) error {
					parts := splitAndTrim(s)
					if len(parts) < 2 {
						return fmt.Errorf("at least 2 files required")
					}
					return nil
				}),
		),
	).WithTheme(ui.Theme())

	if err := form.Run(); err != nil {
		return err
	}

	cfg.Files = splitAndTrim(filesStr)
	return nil
}

func splitAndTrim(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
