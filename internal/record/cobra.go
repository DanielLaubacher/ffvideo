package record

import (
	"github.com/spf13/cobra"

	"github.com/DanielLaubacher/ffvideo/internal/config"
)

// NewCommand returns the "record" cobra.Command with config defaults applied.
func NewCommand(defaults config.RecordDefaults) *cobra.Command {
	cfg := Config{
		Framerate: defaults.Framerate,
		Silent:    defaults.Silent,
		AudioDev:  defaults.AudioDevice,
		Preset:    defaults.Preset,
		CRF:       defaults.CRF,
	}

	cmd := &cobra.Command{
		Use:   "record -o <file> [flags]",
		Short: "Record screen, window, or region",
		Long: `Record screen, window, or selected region using ffmpeg.

Without -d, you will be prompted to select a region.
Install 'slop' (X11) or 'slurp' (Wayland) for click-drag region selection.
Falls back to xwininfo (window-only selection) if neither is available.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Output == "" {
				if err := RunForm(&cfg); err != nil {
					return err
				}
			}
			return Run(cmd.Context(), &cfg)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&cfg.Output, "output", "o", "", "output file (required)")
	f.BoolVarP(&cfg.Desktop, "desktop", "d", cfg.Desktop, "record full desktop (primary monitor)")
	f.BoolVarP(&cfg.Silent, "silent", "s", cfg.Silent, "no audio capture")
	f.IntVarP(&cfg.Framerate, "framerate", "r", cfg.Framerate, "frame rate")
	f.StringVarP(&cfg.AudioDev, "audio", "a", cfg.AudioDev, "audio device override")

	return cmd
}
