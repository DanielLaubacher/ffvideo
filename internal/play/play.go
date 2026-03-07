package play

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// NewCommand returns the "play" cobra.Command.
func NewCommand() *cobra.Command {
	var loop bool

	cmd := &cobra.Command{
		Use:   "play <file>",
		Short: "Play a video with ffplay",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), args[0], loop)
		},
	}

	cmd.Flags().BoolVarP(&loop, "loop", "l", false, "loop playback")
	return cmd
}

func run(ctx context.Context, file string, loop bool) error {
	if _, err := os.Stat(file); err != nil {
		return fmt.Errorf("file not found: %s", file)
	}

	args := []string{"ffplay", "-autoexit"}
	if loop {
		args = append(args, "-loop", "0")
	}
	args = append(args, file)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
