package convert

import "github.com/spf13/cobra"

// NewCommand returns the "convert" cobra.Command.
func NewCommand() *cobra.Command {
	cfg := Config{}

	cmd := &cobra.Command{
		Use:   "convert -i <input> -o <output> [flags]",
		Short: "Convert video format (gif, webp, webm, mp4)",
		Long: `Convert a video to another format. The output format is inferred from the
file extension — no need to specify codecs or filters.

Supported formats:
  .gif   Two-pass palette-based GIF (default: 640px wide, 15fps)
  .webp  Animated WebP (default: 640px wide, 15fps)
  .webm  VP9/Opus WebM
  .mp4   Re-encode h264/AAC

Use --width and --fps to override the defaults.`,
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
	f.StringVarP(&cfg.Output, "output", "o", "", "output file (required, format inferred from extension)")
	f.IntVarP(&cfg.Width, "width", "w", 0, "output width in pixels (0 = format default)")
	f.IntVar(&cfg.FPS, "fps", 0, "output frame rate (0 = format default)")

	return cmd
}
