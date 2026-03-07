package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/DanielLaubacher/ffvideo/internal/config"
	"github.com/DanielLaubacher/ffvideo/internal/configcmd"
	"github.com/DanielLaubacher/ffvideo/internal/convert"
	"github.com/DanielLaubacher/ffvideo/internal/edit"
	"github.com/DanielLaubacher/ffvideo/internal/merge"
	"github.com/DanielLaubacher/ffvideo/internal/play"
	"github.com/DanielLaubacher/ffvideo/internal/record"
)

func main() {
	cfg := config.Load()

	root := &cobra.Command{
		Use:   "ffvideo",
		Short: "Screen recorder and video editor",
		Long:  "Record screen, edit videos, and merge clips using ffmpeg.",
	}

	root.AddCommand(
		record.NewCommand(cfg.Record),
		edit.NewCommand(cfg.Edit),
		merge.NewCommand(),
		convert.NewCommand(),
		play.NewCommand(),
		configcmd.NewCommand(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
