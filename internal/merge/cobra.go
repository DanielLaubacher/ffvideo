package merge

import "github.com/spf13/cobra"

// NewCommand returns the "merge" cobra.Command.
func NewCommand() *cobra.Command {
	cfg := Config{}

	cmd := &cobra.Command{
		Use:   "merge -o <output> <file1> <file2> [...]",
		Short: "Combine multiple videos into one",
		Long: `Concatenate multiple video files using ffmpeg's concat demuxer.

Provide input files as positional arguments, or use --list to read paths from a file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Files = args
			if cfg.Output == "" || (cfg.ListFile == "" && len(cfg.Files) == 0) {
				if err := RunForm(&cfg); err != nil {
					return err
				}
			}
			return Run(cmd.Context(), &cfg)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&cfg.Output, "output", "o", "", "output file (required)")
	f.StringVar(&cfg.ListFile, "list", "", "read input paths from a text file (one per line)")

	return cmd
}
