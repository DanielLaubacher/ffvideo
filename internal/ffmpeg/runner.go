package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// Run executes an ffmpeg command with terminal I/O and signal handling.
// On SIGINT/SIGTERM, it writes "q\n" to ffmpeg's stdin for graceful shutdown.
// Exit code 255 (ffmpeg's Ctrl-C exit) is treated as success.
func Run(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		stdin.Write([]byte("q\n"))
		stdin.Close()
	}()

	err = cmd.Wait()
	signal.Stop(sigCh)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 255 {
				return nil
			}
		}
		return fmt.Errorf("ffmpeg: %w", err)
	}
	return nil
}

// RunSilent executes an ffmpeg command without terminal output.
// Stderr is captured for error reporting. Used for edit pipeline stages.
func RunSilent(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w\n%s", err, output)
	}
	return nil
}
