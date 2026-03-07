package edit

import (
	"github.com/spf13/cobra"

	"github.com/DanielLaubacher/ffvideo/internal/config"
)

// NewCommand returns the "edit" cobra.Command with config defaults applied.
func NewCommand(defaults config.EditDefaults) *cobra.Command {
	cfg := Config{
		AudioVolume: defaults.AudioVolume,
		TextColor:   defaults.TextColor,
		TextSize:    defaults.TextSize,
	}

	cmd := &cobra.Command{
		Use:   "edit -i <input> -o <output> [flags]",
		Short: "Trim, filter, and process a video",
		Long: `Process a video file with one or more operations.
Operations applied in order: trim → normalize → denoise → audio → text.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Input == "" || cfg.Output == "" {
				if err := RunForm(&cfg); err != nil {
					return err
				}
			}
			return Run(cmd.Context(), &cfg)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&cfg.Input, "input", "i", "", "input video file (required)")
	f.StringVarP(&cfg.Output, "output", "o", "", "output video file (required)")
	f.StringVar(&cfg.TrimStart, "trim-start", "", "remove content before TIME (HH:MM:SS or seconds)")
	f.StringVar(&cfg.TrimEnd, "trim-end", "", "remove content after TIME")
	f.BoolVar(&cfg.Normalize, "normalize", false, "normalize audio levels (EBU R128)")
	f.BoolVar(&cfg.Denoise, "denoise", false, "reduce background noise")
	f.StringVar(&cfg.AudioFile, "audio", "", "mix in background audio track")
	f.StringVar(&cfg.AudioVolume, "audio-volume", cfg.AudioVolume, "volume for background audio (0-1)")
	f.StringVar(&cfg.Text, "text", "", "add banner text at top of video")
	f.StringVar(&cfg.TextColor, "text-color", cfg.TextColor, "text color")
	f.IntVar(&cfg.TextSize, "text-size", cfg.TextSize, "text font size")
	f.BoolVar(&cfg.TextBold, "text-bold", false, "use bold font")

	return cmd
}
