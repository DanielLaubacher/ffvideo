package ffmpeg

import "fmt"

// AudioInput represents an ffmpeg audio input source.
type AudioInput struct {
	Format string // "pulse" or "alsa"
	Device string // "default" or "plughw:0,0"
}

// RecordArgs configures a screen recording ffmpeg command.
type RecordArgs struct {
	Width, Height int
	X, Y          int
	Framerate     int
	Audio         *AudioInput // nil = silent
	Preset        string      // x264 preset, e.g. "ultrafast"
	CRF           int         // quality factor, e.g. 18
	Output        string
}

// Record returns ffmpeg args for screen capture.
func Record(a RecordArgs) []string {
	if a.Preset == "" {
		a.Preset = "ultrafast"
	}
	if a.CRF == 0 {
		a.CRF = 18
	}

	args := []string{
		"ffmpeg", "-hide_banner", "-y",
		"-thread_queue_size", "512",
		"-video_size", fmt.Sprintf("%dx%d", a.Width, a.Height),
		"-framerate", fmt.Sprintf("%d", a.Framerate),
		"-f", "x11grab",
		"-i", fmt.Sprintf(":0.0+%d,%d", a.X, a.Y),
	}

	if a.Audio != nil {
		args = append(args, "-thread_queue_size", "1024",
			"-f", a.Audio.Format, "-i", a.Audio.Device)
		args = append(args, "-c:a", "aac")
	}

	args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-preset", a.Preset, "-crf", fmt.Sprintf("%d", a.CRF))
	args = append(args, a.Output)
	return args
}

// TrimArgs configures a video trim operation.
type TrimArgs struct {
	Input  string
	Output string
	Start  string // "" to omit -ss
	End    string // "" to omit -to
}

// Trim returns ffmpeg args for trimming a video.
// -ss is placed before -i for fast input seeking.
func Trim(a TrimArgs) []string {
	args := []string{"ffmpeg", "-hide_banner", "-y"}
	if a.Start != "" {
		args = append(args, "-ss", a.Start)
	}
	args = append(args, "-i", a.Input)
	if a.End != "" {
		args = append(args, "-to", a.End)
	}
	args = append(args, "-c:v", "libx264", "-c:a", "aac", "-map", "0:v", "-map", "0:a?")
	args = append(args, a.Output)
	return args
}

// TrimCopy returns ffmpeg args for trimming with stream copy (no re-encode).
// Much faster than Trim() but may have slight inaccuracy at keyframe boundaries.
func TrimCopy(a TrimArgs) []string {
	args := []string{"ffmpeg", "-hide_banner", "-y"}
	if a.Start != "" {
		args = append(args, "-ss", a.Start)
	}
	args = append(args, "-i", a.Input)
	if a.End != "" {
		args = append(args, "-to", a.End)
	}
	args = append(args, "-c", "copy", "-map", "0:v", "-map", "0:a?")
	args = append(args, a.Output)
	return args
}

// NormalizeArgs configures audio normalization.
type NormalizeArgs struct {
	Input  string
	Output string
}

// Normalize returns ffmpeg args for EBU R128 loudness normalization.
func Normalize(a NormalizeArgs) []string {
	return []string{
		"ffmpeg", "-hide_banner", "-y", "-i", a.Input,
		"-af", "loudnorm=I=-16:TP=-1.5:LRA=11",
		"-c:v", "copy",
		a.Output,
	}
}

// DenoiseArgs configures noise reduction.
type DenoiseArgs struct {
	Input  string
	Output string
}

// Denoise returns ffmpeg args for FFT-based noise reduction.
func Denoise(a DenoiseArgs) []string {
	return []string{
		"ffmpeg", "-hide_banner", "-y", "-i", a.Input,
		"-af", "afftdn=nf=-25",
		"-c:v", "copy",
		a.Output,
	}
}

// MixAudioArgs configures background audio mixing.
type MixAudioArgs struct {
	VideoInput string
	AudioInput string
	Output     string
	Volume     string  // e.g. "0.2"
	Duration   float64 // video duration in seconds
	HasAudio   bool    // whether the video has an existing audio stream
}

// MixAudio returns ffmpeg args for mixing background audio into a video.
func MixAudio(a MixAudioArgs) []string {
	args := []string{
		"ffmpeg", "-hide_banner", "-y",
		"-i", a.VideoInput,
		"-i", a.AudioInput,
	}

	if a.HasAudio {
		args = append(args, "-filter_complex",
			fmt.Sprintf("[1:a]aloop=loop=-1:size=2e+09,asetpts=N/SR/TB,volume=%s[bg];[0:a][bg]amix=inputs=2:duration=first[a]", a.Volume))
		args = append(args, "-map", "0:v", "-map", "[a]")
	} else {
		args = append(args, "-filter_complex",
			fmt.Sprintf("[1:a]aloop=loop=-1:size=2e+09,asetpts=N/SR/TB,volume=%s[a]", a.Volume))
		args = append(args, "-map", "0:v", "-map", "[a]")
	}

	args = append(args, "-t", fmt.Sprintf("%.6f", a.Duration), "-c:v", "copy", "-c:a", "aac")
	args = append(args, a.Output)
	return args
}

// DrawTextArgs configures banner text overlay.
type DrawTextArgs struct {
	Input    string
	Output   string
	TextFile string // path to file containing the text
	FontFile string // path to font file, may be ""
	Color    string
	Size     int
}

// DrawText returns ffmpeg args for adding a text banner.
func DrawText(a DrawTextArgs) []string {
	filter := fmt.Sprintf(
		"drawtext=textfile='%s':fontcolor=%s:fontsize=%d:x=(w-text_w)/2:y=10:box=1:boxcolor=black@0.5:boxborderw=5",
		a.TextFile, a.Color, a.Size)
	if a.FontFile != "" {
		filter += fmt.Sprintf(":fontfile='%s'", a.FontFile)
	}

	return []string{
		"ffmpeg", "-hide_banner", "-y", "-i", a.Input,
		"-vf", filter,
		"-c:a", "copy",
		a.Output,
	}
}

// ConvertArgs configures format conversion with output-type-aware defaults.
type ConvertArgs struct {
	Input   string
	Output  string
	Width   int // 0 = keep original
	FPS     int // 0 = keep original
}

// GIF returns ffmpeg args for single-pass palette-based GIF conversion.
// Uses the split filter to generate and apply a palette in one pass.
func GIF(input, output string, width, fps int) []string {
	vf := buildScaleFPS(width, fps)
	if vf != "" {
		vf += ","
	}
	vf += "split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse"
	return []string{
		"ffmpeg", "-hide_banner", "-y", "-i", input,
		"-vf", vf,
		"-loop", "0",
		output,
	}
}

// ConvertToWebM returns ffmpeg args for VP9 WebM conversion.
func ConvertToWebM(a ConvertArgs) []string {
	args := []string{"ffmpeg", "-hide_banner", "-y", "-i", a.Input}

	vf := buildScaleFPS(a.Width, a.FPS)
	if vf != "" {
		args = append(args, "-vf", vf)
	}

	args = append(args, "-c:v", "libvpx-vp9", "-crf", "30", "-b:v", "0")
	args = append(args, "-c:a", "libopus")
	args = append(args, a.Output)
	return args
}

// ConvertToWebP returns ffmpeg args for animated WebP conversion.
func ConvertToWebP(a ConvertArgs) []string {
	args := []string{"ffmpeg", "-hide_banner", "-y", "-i", a.Input}

	vf := buildScaleFPS(a.Width, a.FPS)
	if vf != "" {
		args = append(args, "-vf", vf)
	}

	args = append(args, "-c:v", "libwebp", "-compression_level", "6",
		"-quality", "80", "-loop", "0")
	args = append(args, "-an")
	args = append(args, a.Output)
	return args
}

func buildScaleFPS(width, fps int) string {
	var parts []string
	if fps > 0 {
		parts = append(parts, fmt.Sprintf("fps=%d", fps))
	}
	if width > 0 {
		// -2 maintains aspect ratio and rounds to even; lanczos for sharp downscaling
		parts = append(parts, fmt.Sprintf("scale=%d:-2:flags=lanczos", width))
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ","
		}
		result += p
	}
	return result
}

// ConcatArgs configures video concatenation.
type ConcatArgs struct {
	ListFile string
	Output   string
}

// Concat returns ffmpeg args for concatenating videos with stream copy.
func Concat(a ConcatArgs) []string {
	return []string{
		"ffmpeg", "-hide_banner", "-y",
		"-f", "concat", "-safe", "0",
		"-i", a.ListFile,
		"-c", "copy",
		a.Output,
	}
}

// ConcatFilter returns ffmpeg args for concatenating videos with different
// codecs or resolutions. This re-encodes all inputs to a common format.
func ConcatFilter(inputs []string, output string) []string {
	args := []string{"ffmpeg", "-hide_banner", "-y"}
	for _, f := range inputs {
		args = append(args, "-i", f)
	}

	var filter string
	for i := range inputs {
		filter += fmt.Sprintf("[%d:v][%d:a]", i, i)
	}
	filter += fmt.Sprintf("concat=n=%d:v=1:a=1[outv][outa]", len(inputs))

	args = append(args, "-filter_complex", filter)
	args = append(args, "-map", "[outv]", "-map", "[outa]")
	args = append(args, "-c:v", "libx264", "-c:a", "aac", "-movflags", "+faststart")
	args = append(args, output)
	return args
}
