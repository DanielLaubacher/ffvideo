#!/bin/bash
set -euo pipefail

SCRIPT_NAME="$(basename "$0")"

# ---- Utilities ----

die() {
    echo "Error: $*" >&2
    exit 1
}

require_cmd() {
    command -v "$1" &>/dev/null || die "'$1' is not installed."
}

round_even() {
    local n="$1"
    echo $(( n + (n % 2) ))
}

# ---- Help ----

help() {
    local cmd="${1:-}"
    case "$cmd" in
        record) record_help ;;
        edit)   edit_help ;;
        merge)  merge_help ;;
        *)
            cat <<EOF
Usage: $SCRIPT_NAME <command> [options]

Commands:
  record    Record screen, window, or region
  edit      Trim, filter, and process a video
  merge     Combine multiple videos into one
  help      Show help for a command

Run '$SCRIPT_NAME help <command>' for command details.
EOF
            ;;
    esac
}

# ---- Record ----

record_help() {
    cat <<EOF
Usage: $SCRIPT_NAME record -o <file> [options]

Record screen, window, or selected region.

Options:
  -o FILE       Output file (required)
  -d            Record full desktop (primary monitor)
  -s            Silent mode (no audio capture)
  -r RATE       Frame rate (default: 30)
  -a DEVICE     Audio device string for ffmpeg
                (auto-detected from PulseAudio/ALSA if omitted)

Without -d, you will be prompted to select a region.
Install 'slop' (X11) or 'slurp' (Wayland) for click-drag region selection.
Falls back to xwininfo (window-only selection) if neither is available.
EOF
}

detect_audio() {
    if command -v pactl &>/dev/null; then
        echo "-f pulse -i default"
        return
    fi
    local line
    if line=$(arecord -l 2>/dev/null | grep -m1 '^card'); then
        local card device
        card=$(echo "$line" | sed 's/^card \([0-9]*\).*/\1/')
        device=$(echo "$line" | sed 's/.*device \([0-9]*\).*/\1/')
        echo "-f alsa -i plughw:${card},${device}"
        return
    fi
}

screen_geometry() {
    require_cmd xrandr
    local geom
    geom=$(xrandr --current | grep -oP 'primary \K\d+x\d+\+\d+\+\d+') || \
        geom=$(xrandr --current | grep -oP '\d+x\d+\+\d+\+\d+' | head -1) || \
        die "Could not detect screen geometry."
    local size="${geom%%+*}"
    local rest="${geom#*+}"
    echo "${size%x*} ${size#*x} ${rest%%+*} ${rest#*+}"
}

select_region() {
    if command -v slop &>/dev/null; then
        # X11: drag to select any region
        slop -f '%w %h %x %y' || die "Selection cancelled."
    elif command -v slurp &>/dev/null; then
        # Wayland: drag to select any region
        local geom
        geom=$(slurp) || die "Selection cancelled."
        # slurp outputs "X,Y WxH" — reorder to "W H X Y"
        local x y w h
        x="${geom%%,*}"
        y="${geom#*,}"; y="${y%% *}"
        w="${geom##* }"; w="${w%%x*}"
        h="${geom##*x}"
        echo "$w $h $x $y"
    elif command -v xwininfo &>/dev/null; then
        # X11 fallback: click to select a window
        local info
        info=$(xwininfo) || die "Selection cancelled."
        local x y w h
        x=$(echo "$info" | awk '/Absolute upper-left X/ { print $4 }')
        y=$(echo "$info" | awk '/Absolute upper-left Y/ { print $4 }')
        w=$(echo "$info" | awk '/Width/ { print $2 }')
        h=$(echo "$info" | awk '/Height/ { print $2 }')
        echo "$w $h $x $y"
    else
        die "Install 'slop' (X11) or 'slurp' (Wayland) for region selection."
    fi
}

record() {
    local output="" desktop=false silent=false framerate=30 audio_device=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            -o) output="${2:?"-o requires a filename"}"; shift 2 ;;
            -d) desktop=true; shift ;;
            -s) silent=true; shift ;;
            -r) framerate="${2:?"-r requires a number"}"; shift 2 ;;
            -a) audio_device="$2"; shift 2 ;;
            -h|--help) record_help; return 0 ;;
            *) die "Unknown option: $1 (see '$SCRIPT_NAME help record')" ;;
        esac
    done

    [[ -n "$output" ]] || die "Output file required (-o)."
    require_cmd ffmpeg

    # Audio
    local audio_args=()
    if [[ "$silent" == false ]]; then
        local detected=""
        if [[ -n "$audio_device" ]]; then
            detected="$audio_device"
        else
            detected=$(detect_audio) || true
        fi
        if [[ -n "$detected" ]]; then
            read -ra audio_args <<< "$detected"
        else
            echo "Warning: No audio device found, recording without audio." >&2
        fi
    fi

    # Video geometry
    local w h x y
    if [[ "$desktop" == true ]]; then
        read -r w h x y <<< "$(screen_geometry)"
    else
        echo "Select a region or window..." >&2
        read -r w h x y <<< "$(select_region)"
    fi
    w=$(round_even "$w")
    h=$(round_even "$h")

    # Build ffmpeg command as array for safe execution
    local cmd=(ffmpeg -y
        -video_size "${w}x${h}"
        -framerate "$framerate"
        -f x11grab
        -i ":0.0+${x},${y}"
    )

    if [[ ${#audio_args[@]} -gt 0 ]]; then
        cmd+=("${audio_args[@]}" -c:a aac)
    fi

    cmd+=(-c:v libx264 -preset ultrafast -crf 18 "$output")

    trap 'echo ""; echo "Recording stopped: $output"' INT TERM

    echo "Recording ${w}x${h}+${x},${y} @ ${framerate}fps" >&2
    echo "Press q or Ctrl+C to stop." >&2

    local rc=0
    "${cmd[@]}" || rc=$?
    # ffmpeg exits 255 on Ctrl+C after writing the file trailer
    if (( rc != 0 && rc != 255 )); then
        die "ffmpeg failed (exit code: $rc)"
    fi

    echo "Saved: $output"
    echo "Play:  ffplay \"$output\""
}

# ---- Edit ----

edit_help() {
    cat <<EOF
Usage: $SCRIPT_NAME edit -i <input> -o <output> [options]

Process a video file with one or more operations.
Operations applied in order: trim -> normalize -> denoise -> audio -> text.

Options:
  -i FILE              Input video file (required)
  -o FILE              Output video file (required)
  --trim-start TIME    Remove content before TIME (HH:MM:SS or seconds)
  --trim-end TIME      Remove content after TIME
  --normalize          Normalize audio levels (EBU R128)
  --denoise            Reduce background noise
  --audio FILE         Mix in background audio track
  --audio-volume LEVEL Volume for background audio (default: 0.2, range: 0-1)
  --text TEXT          Add banner text at top of video
  --text-color COLOR   Text color (default: white)
  --text-size SIZE     Text font size (default: 24)
  --text-bold          Use bold font

Examples:
  $SCRIPT_NAME edit -i raw.mp4 -o final.mp4 --trim-start 00:00:05 --trim-end 00:02:00
  $SCRIPT_NAME edit -i raw.mp4 -o final.mp4 --normalize --denoise
  $SCRIPT_NAME edit -i raw.mp4 -o final.mp4 --text "Demo" --text-color yellow
EOF
}

edit() {
    local input="" output=""
    local trim_start="" trim_end=""
    local normalize=false denoise=false
    local audio_file="" audio_volume="0.2"
    local text="" text_color="white" text_size=24 text_bold=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            -i|--input)       input="$2"; shift 2 ;;
            -o|--output)      output="$2"; shift 2 ;;
            --trim-start)     trim_start="$2"; shift 2 ;;
            --trim-end)       trim_end="$2"; shift 2 ;;
            --normalize)      normalize=true; shift ;;
            --denoise)        denoise=true; shift ;;
            --audio)          audio_file="$2"; shift 2 ;;
            --audio-volume)   audio_volume="$2"; shift 2 ;;
            --text)           text="$2"; shift 2 ;;
            --text-color)     text_color="$2"; shift 2 ;;
            --text-size)      text_size="$2"; shift 2 ;;
            --text-bold)      text_bold=true; shift ;;
            -h|--help)        edit_help; return 0 ;;
            *) die "Unknown option: $1 (see '$SCRIPT_NAME help edit')" ;;
        esac
    done

    [[ -n "$input" ]]  || die "Input file required (-i)."
    [[ -n "$output" ]] || die "Output file required (-o)."
    [[ -f "$input" ]]  || die "Input file not found: $input"
    require_cmd ffmpeg
    require_cmd ffprobe

    local tmpdir
    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    local current="$input"
    local step=0

    # Trim
    if [[ -n "$trim_start" || -n "$trim_end" ]]; then
        local trimmed="${tmpdir}/$((++step))_trimmed.mp4"
        local trim_args=(-y -i "$current")
        [[ -n "$trim_start" ]] && trim_args+=(-ss "$trim_start")
        [[ -n "$trim_end" ]]   && trim_args+=(-to "$trim_end")
        trim_args+=(-c:v libx264 -c:a aac -map 0:v -map 0:a? "$trimmed")

        echo "Trimming${trim_start:+ from $trim_start}${trim_end:+ to $trim_end}..."
        ffmpeg "${trim_args[@]}"
        current="$trimmed"
    fi

    # Normalize audio
    if [[ "$normalize" == true ]]; then
        local normalized="${tmpdir}/$((++step))_normalized.mp4"
        echo "Normalizing audio (EBU R128)..."
        ffmpeg -y -i "$current" -af "loudnorm=I=-16:TP=-1.5:LRA=11" -c:v copy "$normalized"
        current="$normalized"
    fi

    # Denoise
    if [[ "$denoise" == true ]]; then
        local denoised="${tmpdir}/$((++step))_denoised.mp4"
        echo "Reducing noise..."
        ffmpeg -y -i "$current" -af "afftdn=nf=-25" -c:v copy "$denoised"
        current="$denoised"
    fi

    # Background audio
    if [[ -n "$audio_file" ]]; then
        [[ -f "$audio_file" ]] || die "Audio file not found: $audio_file"
        local with_audio="${tmpdir}/$((++step))_audio.mp4"
        local duration
        duration=$(ffprobe -v error -show_entries format=duration \
            -of default=noprint_wrappers=1:nokey=1 "$current")

        echo "Adding background audio (volume: $audio_volume)..."

        # Check if input has an audio stream
        local has_audio
        has_audio=$(ffprobe -v error -select_streams a -show_entries stream=index \
            -of csv=p=0 "$current" | head -1) || true

        if [[ -n "$has_audio" ]]; then
            ffmpeg -y -i "$current" -i "$audio_file" -filter_complex \
                "[1:a]aloop=loop=-1:size=2e+09,asetpts=N/SR/TB,volume=${audio_volume}[bg]; \
                 [0:a][bg]amix=inputs=2:duration=first[a]" \
                -map 0:v -map "[a]" -t "$duration" -c:v copy -c:a aac "$with_audio"
        else
            ffmpeg -y -i "$current" -i "$audio_file" -filter_complex \
                "[1:a]aloop=loop=-1:size=2e+09,asetpts=N/SR/TB,volume=${audio_volume}[a]" \
                -map 0:v -map "[a]" -t "$duration" -c:v copy -c:a aac "$with_audio"
        fi
        current="$with_audio"
    fi

    # Banner text
    if [[ -n "$text" ]]; then
        local with_text="${tmpdir}/$((++step))_text.mp4"

        # Write text to a file to avoid filter string escaping issues
        local textfile="${tmpdir}/banner.txt"
        printf '%s' "$text" > "$textfile"

        # Use fontconfig to find an appropriate font
        local font=""
        if command -v fc-match &>/dev/null; then
            if [[ "$text_bold" == true ]]; then
                font=$(fc-match --format='%{file}' "sans:bold" 2>/dev/null) || true
            else
                font=$(fc-match --format='%{file}' "sans" 2>/dev/null) || true
            fi
        fi

        local drawtext="textfile='${textfile}':fontcolor=${text_color}:fontsize=${text_size}"
        drawtext+=":x=(w-text_w)/2:y=10:box=1:boxcolor=black@0.5:boxborderw=5"
        [[ -n "$font" ]] && drawtext+=":fontfile='${font}'"

        echo "Adding text: '$text'..."
        ffmpeg -y -i "$current" -vf "drawtext=${drawtext}" -c:a copy "$with_text"
        current="$with_text"
    fi

    cp "$current" "$output"
    echo "Done: $output"
}

# ---- Merge ----

merge_help() {
    cat <<EOF
Usage: $SCRIPT_NAME merge -o <output> <file1> <file2> [...]
       $SCRIPT_NAME merge -o <output> --list <file>

Concatenate multiple video files.

Options:
  -o FILE        Output file (required)
  --list FILE    Read input paths from a text file (one per line)

Examples:
  $SCRIPT_NAME merge -o combined.mp4 part1.mp4 part2.mp4 part3.mp4
  $SCRIPT_NAME merge -o combined.mp4 --list videos.txt
EOF
}

merge() {
    local output="" list_file="" files=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            -o)     output="$2"; shift 2 ;;
            --list) list_file="$2"; shift 2 ;;
            -h|--help) merge_help; return 0 ;;
            -*)     die "Unknown option: $1 (see '$SCRIPT_NAME help merge')" ;;
            *)      files+=("$1"); shift ;;
        esac
    done

    [[ -n "$output" ]] || die "Output file required (-o)."
    require_cmd ffmpeg

    local tmpdir
    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    local concat="${tmpdir}/concat.txt"

    if [[ -n "$list_file" ]]; then
        [[ -f "$list_file" ]] || die "List file not found: $list_file"
        while IFS= read -r f || [[ -n "$f" ]]; do
            f="${f#"${f%%[![:space:]]*}"}"
            f="${f%"${f##*[![:space:]]}"}"
            [[ -z "$f" || "$f" == \#* ]] && continue
            [[ -f "$f" ]] || { echo "Warning: '$f' not found, skipping." >&2; continue; }
            echo "file '$(realpath "$f")'" >> "$concat"
        done < "$list_file"
    else
        [[ ${#files[@]} -ge 2 ]] || die "At least 2 input files required."
        for f in "${files[@]}"; do
            [[ -f "$f" ]] || die "File not found: $f"
            echo "file '$(realpath "$f")'" >> "$concat"
        done
    fi

    [[ -s "$concat" ]] || die "No valid input files."

    echo "Merging $(wc -l < "$concat") files..."
    ffmpeg -y -f concat -safe 0 -i "$concat" -c copy "$output"
    echo "Done: $output"
}

# ---- Dispatch ----

[[ $# -gt 0 ]] || { help; exit 0; }

cmd="$1"
if declare -F "$cmd" &>/dev/null; then
    "$cmd" "${@:2}"
else
    echo "Error: Command '$cmd' is not defined." >&2
    help
    exit 1
fi
