package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all persistent user preferences.
type Config struct {
	Record RecordDefaults `toml:"record"`
	Edit   EditDefaults   `toml:"edit"`
}

// RecordDefaults are default values for the record subcommand.
type RecordDefaults struct {
	Framerate   int    `toml:"framerate"`
	Silent      bool   `toml:"silent"`
	AudioDevice string `toml:"audio_device"`
	Preset      string `toml:"preset"`
	CRF         int    `toml:"crf"`
}

// EditDefaults are default values for the edit subcommand.
type EditDefaults struct {
	AudioVolume string `toml:"audio_volume"`
	TextColor   string `toml:"text_color"`
	TextSize    int    `toml:"text_size"`
}

// Defaults returns a Config with sensible defaults.
func Defaults() Config {
	return Config{
		Record: RecordDefaults{
			Framerate: 30,
			Preset:    "ultrafast",
			CRF:       18,
		},
		Edit: EditDefaults{
			AudioVolume: "0.2",
			TextColor:   "white",
			TextSize:    24,
		},
	}
}

// Path returns the config file path: ~/.config/ffvideo/config.toml
func Path() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "ffvideo", "config.toml")
}

// Load reads the config file. Returns defaults if the file doesn't exist.
func Load() Config {
	cfg := Defaults()
	path := Path()

	if _, err := os.Stat(path); err != nil {
		return cfg
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse %s: %v\n", path, err)
		return Defaults()
	}
	return cfg
}

// Save writes the config to disk, creating the directory if needed.
func Save(cfg Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
