package ffmpeg

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Duration returns the duration of a media file in seconds.
func Duration(path string) (float64, error) {
	out, err := exec.Command(
		"ffprobe", "-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration: %w", err)
	}
	return strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
}

// VideoCodec returns the codec name of the first video stream.
func VideoCodec(path string) (string, error) {
	out, err := exec.Command(
		"ffprobe", "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "csv=p=0",
		path,
	).Output()
	if err != nil {
		return "", fmt.Errorf("ffprobe codec: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Resolution returns width and height of the first video stream.
func Resolution(path string) (int, int, error) {
	out, err := exec.Command(
		"ffprobe", "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0",
		path,
	).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe resolution: %w", err)
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected resolution output: %q", string(out))
	}
	w, _ := strconv.Atoi(parts[0])
	h, _ := strconv.Atoi(parts[1])
	return w, h, nil
}

// HasAudioStream returns true if the file contains an audio stream.
func HasAudioStream(path string) (bool, error) {
	out, err := exec.Command(
		"ffprobe", "-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=index",
		"-of", "csv=p=0",
		path,
	).Output()
	if err != nil {
		return false, fmt.Errorf("ffprobe audio check: %w", err)
	}
	return strings.TrimSpace(string(out)) != "", nil
}
