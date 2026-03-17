package detect

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/DanielLaubacher/ffvideo/internal/ffmpeg"
)

// DetectAudio finds an available audio capture device.
// Checks PulseAudio/PipeWire first, then falls back to ALSA.
// Returns nil if no audio device is found.
func DetectAudio() *ffmpeg.AudioInput {
	if _, err := exec.LookPath("pactl"); err == nil {
		return &ffmpeg.AudioInput{Format: "pulse", Device: "default"}
	}

	out, err := exec.Command("arecord", "-l").Output()
	if err != nil {
		return nil
	}

	re := regexp.MustCompile(`^card (\d+):.*device (\d+):`)
	for line := range strings.SplitSeq(string(out), "\n") {
		m := re.FindStringSubmatch(line)
		if m != nil {
			return &ffmpeg.AudioInput{
				Format: "alsa",
				Device: "plughw:" + m[1] + "," + m[2],
			}
		}
	}
	return nil
}
