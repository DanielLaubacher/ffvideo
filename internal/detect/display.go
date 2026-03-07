package detect

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Geometry represents a screen region.
type Geometry struct {
	Width, Height int
	X, Y          int
}

// DisplayServer returns "wayland", "x11", or "unknown".
func DisplayServer() string {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "wayland"
	}
	if os.Getenv("DISPLAY") != "" {
		return "x11"
	}
	if t := os.Getenv("XDG_SESSION_TYPE"); t != "" {
		return t
	}
	return "unknown"
}

// RoundEven rounds n up to the nearest even number.
func RoundEven(n int) int {
	return n + (n % 2)
}

// ScreenGeometry returns the primary monitor's geometry via xrandr.
func ScreenGeometry() (Geometry, error) {
	out, err := exec.Command("xrandr", "--current").Output()
	if err != nil {
		return Geometry{}, fmt.Errorf("xrandr: %w", err)
	}

	lines := string(out)

	// Try primary monitor first: "primary 1920x1080+0+0"
	re := regexp.MustCompile(`primary (\d+)x(\d+)\+(\d+)\+(\d+)`)
	m := re.FindStringSubmatch(lines)
	if m == nil {
		// Fall back to first connected output: "1920x1080+0+0"
		re = regexp.MustCompile(`(\d+)x(\d+)\+(\d+)\+(\d+)`)
		m = re.FindStringSubmatch(lines)
	}
	if m == nil {
		return Geometry{}, fmt.Errorf("could not detect screen geometry from xrandr output")
	}

	w, _ := strconv.Atoi(m[1])
	h, _ := strconv.Atoi(m[2])
	x, _ := strconv.Atoi(m[3])
	y, _ := strconv.Atoi(m[4])
	return Geometry{Width: w, Height: h, X: x, Y: y}, nil
}

// SelectRegion prompts the user to select a screen region.
// Tries slop (X11), then slurp (Wayland), then xwininfo (X11 window fallback).
func SelectRegion() (Geometry, error) {
	if path, err := exec.LookPath("slop"); err == nil {
		return runSlop(path)
	}
	if path, err := exec.LookPath("slurp"); err == nil {
		return runSlurp(path)
	}
	if path, err := exec.LookPath("xwininfo"); err == nil {
		return runXwininfo(path)
	}
	return Geometry{}, fmt.Errorf("install 'slop' (X11) or 'slurp' (Wayland) for region selection")
}

// slop outputs: "W H X Y" with format '%w %h %x %y'
func runSlop(path string) (Geometry, error) {
	out, err := exec.Command(path, "-f", "%w %h %x %y").Output()
	if err != nil {
		return Geometry{}, fmt.Errorf("selection cancelled")
	}
	return parseWHXY(strings.TrimSpace(string(out)))
}

// slurp outputs: "X,Y WxH"
func runSlurp(path string) (Geometry, error) {
	out, err := exec.Command(path).Output()
	if err != nil {
		return Geometry{}, fmt.Errorf("selection cancelled")
	}
	s := strings.TrimSpace(string(out))

	// Parse "X,Y WxH"
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return Geometry{}, fmt.Errorf("unexpected slurp output: %q", s)
	}

	xy := strings.SplitN(parts[0], ",", 2)
	wh := strings.SplitN(parts[1], "x", 2)
	if len(xy) != 2 || len(wh) != 2 {
		return Geometry{}, fmt.Errorf("unexpected slurp output: %q", s)
	}

	x, _ := strconv.Atoi(xy[0])
	y, _ := strconv.Atoi(xy[1])
	w, _ := strconv.Atoi(wh[0])
	h, _ := strconv.Atoi(wh[1])
	return Geometry{Width: w, Height: h, X: x, Y: y}, nil
}

// xwininfo: click to select a window, parse geometry from output.
func runXwininfo(path string) (Geometry, error) {
	cmd := exec.Command(path)
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return Geometry{}, fmt.Errorf("selection cancelled")
	}
	lines := string(out)

	x, err := parseXwininfoField(lines, `Absolute upper-left X:\s+(\d+)`)
	if err != nil {
		return Geometry{}, err
	}
	y, err := parseXwininfoField(lines, `Absolute upper-left Y:\s+(\d+)`)
	if err != nil {
		return Geometry{}, err
	}
	w, err := parseXwininfoField(lines, `Width:\s+(\d+)`)
	if err != nil {
		return Geometry{}, err
	}
	h, err := parseXwininfoField(lines, `Height:\s+(\d+)`)
	if err != nil {
		return Geometry{}, err
	}
	return Geometry{Width: w, Height: h, X: x, Y: y}, nil
}

func parseXwininfoField(output, pattern string) (int, error) {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(output)
	if m == nil {
		return 0, fmt.Errorf("xwininfo: could not parse field %q", pattern)
	}
	return strconv.Atoi(m[1])
}

func parseWHXY(s string) (Geometry, error) {
	parts := strings.Fields(s)
	if len(parts) != 4 {
		return Geometry{}, fmt.Errorf("unexpected geometry output: %q", s)
	}
	w, _ := strconv.Atoi(parts[0])
	h, _ := strconv.Atoi(parts[1])
	x, _ := strconv.Atoi(parts[2])
	y, _ := strconv.Atoi(parts[3])
	return Geometry{Width: w, Height: h, X: x, Y: y}, nil
}
