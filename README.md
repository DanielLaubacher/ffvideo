# ffvideo

A Linux screen recorder and video editor built on ffmpeg. Single binary, interactive TUI or CLI flags.

## Install

Requires: `ffmpeg`, `ffprobe`, and one of `slop` (X11) / `slurp` (Wayland) / `xwininfo` for region selection.

```bash
go install github.com/DanielLaubacher/ffvideo/cmd/ffvideo@latest
```

Or build from source:

```bash
make build        # → bin/ffvideo
make install      # → $GOPATH/bin/ffvideo
```

## Commands

### record

Record screen, window, or selected region.

```bash
ffvideo record -o demo.mp4                  # select region interactively
ffvideo record -o demo.mp4 -d               # full desktop
ffvideo record -o demo.mp4 -d -s            # desktop, no audio
ffvideo record -o demo.gif -d -s            # records mp4, auto-converts to GIF
ffvideo record                               # interactive TUI form
```

### edit

Pipeline of operations on a video: trim, normalize, denoise, background audio, text overlay.

```bash
ffvideo edit -i raw.mp4 -o final.mp4 --trim-start 00:00:05 --trim-end 00:02:00
ffvideo edit -i raw.mp4 -o final.mp4 --normalize --denoise
ffvideo edit -i raw.mp4 -o final.mp4 --text "Demo" --text-color yellow --text-bold
ffvideo edit                                 # interactive TUI form
```

### convert

Convert between formats. Output format inferred from file extension.

```bash
ffvideo convert -i recording.mp4 -o output.gif         # palette-based GIF (640px, 15fps)
ffvideo convert -i recording.mp4 -o output.webp         # animated WebP
ffvideo convert -i recording.mp4 -o output.webm         # VP9/Opus
ffvideo convert -i recording.mp4 -o smaller.mp4 --width 1280
```

### merge

Concatenate multiple videos.

```bash
ffvideo merge -o combined.mp4 part1.mp4 part2.mp4 part3.mp4
ffvideo merge -o combined.mp4 --list videos.txt
```

### play

Play a video with ffplay.

```bash
ffvideo play demo.mp4
ffvideo play -l demo.gif        # loop
```

### config

Interactive wizard to set default preferences (framerate, audio device, encoding quality, etc.).

```bash
ffvideo config
```

Saves to `~/.config/ffvideo/config.toml`. All settings are overridable by flags.

## Example: slide-based presentation

Record individual slides, add audio, then stitch them together.

```bash
# 1. Record each slide silently
ffvideo record -o slide1.mp4 -s
ffvideo record -o slide2.mp4 -s
ffvideo record -o slide3.mp4 -s

# 2. Add narration to each
ffvideo edit -i slide1.mp4 -o slide1_final.mp4 --audio narration1.mp3 --trim-start 00:00:02
ffvideo edit -i slide2.mp4 -o slide2_final.mp4 --audio narration2.mp3 --normalize
ffvideo edit -i slide3.mp4 -o slide3_final.mp4 --audio narration3.mp3 --normalize

# 3. Merge into one video
ffvideo merge -o presentation.mp4 slide1_final.mp4 slide2_final.mp4 slide3_final.mp4

# 4. Convert for sharing
ffvideo convert -i presentation.mp4 -o presentation.gif
```

Or merge first, then add one continuous background track:

```bash
ffvideo merge -o raw.mp4 slide1.mp4 slide2.mp4 slide3.mp4
ffvideo edit -i raw.mp4 -o presentation.mp4 --audio music.mp3 --audio-volume 0.2 --normalize
```

## Interactive mode

Run any command without required flags to get a TUI form powered by [Charm](https://charm.sh).

## Dependencies

Runtime: `ffmpeg`, `ffprobe`, `slop`/`slurp`/`xwininfo`

Build: Go 1.22+
