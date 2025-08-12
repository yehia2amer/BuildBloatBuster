package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	ScanPaths      []string `koanf:"scanPaths"`
	IncludeNames   []string `koanf:"includeNames"`
	ExcludeNames   []string `koanf:"excludeNames"`
	ExcludePaths   []string `koanf:"excludePaths"`
	MinSizeMB      int      `koanf:"minSizeMB"`
	MaxDepth       int      `koanf:"maxDepth"`
	FollowSymlinks bool     `koanf:"followSymlinks"`
	Concurrency    int      `koanf:"concurrency"`
	Delete         struct {
		Mode          string `koanf:"mode"`
		QuarantineDir string `koanf:"quarantineDir"`
		RetentionDays int    `koanf:"retentionDays"`
	} `koanf:"delete"`
	Output struct {
		Format string `koanf:"format"`
		SortBy string `koanf:"sortBy"`
	} `koanf:"output"`
}

// GetDefaults returns the default configuration
func GetDefaults() Config {
	homeDir, _ := os.UserHomeDir()
	quarantineDir := filepath.Join(homeDir, ".cache", "BuildBloatBuster", "trash")

	config := Config{
		ScanPaths: []string{"."},
		IncludeNames: []string{
			"node_modules",
			".venv",
			"venv",
			".tox",
			".pytest_cache",
			"__pycache__",
			".mypy_cache",
			".ruff_cache",
			".parcel-cache",
			".next",
			".nuxt",
			".svelte-kit",
			".turbo",
			".cache",
			"dist",
			"build",
			"out",
			".gradle",
			"target",
			".terraform",
			".serverless",
			"Pods",
			"Carthage/Build",
			"vendor/bundle",
			"vendor",
		},
		ExcludeNames: []string{
			"src", "lib", "source", "Sources", "include",
		},
		ExcludePaths:   getDefaultExcludePaths(homeDir),
		MinSizeMB:      10,
		MaxDepth:       8,
		FollowSymlinks: false,
		Concurrency:    runtime.NumCPU() * 2,
	}

	config.Delete.Mode = "quarantine"
	config.Delete.QuarantineDir = quarantineDir
	config.Delete.RetentionDays = 14

	config.Output.Format = "table"
	config.Output.SortBy = "size"

	return config
}

// GetProtectedPaths returns a list of critical system paths that should never be scanned.
func GetProtectedPaths() []string {
	paths := []string{"/", "/System", "/Library", "/Applications", "/usr", "/bin", "/sbin", "/var", "/etc", "/opt", "/proc", "/dev", "/sys", "/boot", "/root"}

	// On Windows
	if runtime.GOOS == "windows" {
		paths = append(paths, "C:\\", "C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)")
	}

	return paths
}

// getDefaultExcludePaths returns platform-specific default exclude paths
func getDefaultExcludePaths(homeDir string) []string {
	paths := []string{
		"/Applications",
		"/Library",
		"/System",
		"/usr",
		"/bin",
		"/sbin",
		"/opt",
		"/dev",
		"/Volumes",
		"/private",
		"/etc",
		"/var",
		"/tmp",
		"/cores",
	}

	if homeDir != "" {
		paths = append(paths,
			filepath.Join(homeDir, ".cache"),
			filepath.Join(homeDir, "Library"),
			filepath.Join(homeDir, ".Trash"),
			filepath.Join(homeDir, "Applications"),
		)
	}

	return paths
}

// LoadConfig loads configuration from file and merges with defaults
func LoadConfig(path string) (Config, error) {
	// Start with defaults
	config := GetDefaults()

	// Try to load from file
	k := koanf.New(".")
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return config, err // Return defaults with error
	}

	// Merge file config over defaults
	if err := k.Unmarshal("", &config); err != nil {
		return config, err
	}

	return config, nil
}

// LoadConfigWithDefaults loads config or returns defaults if file doesn't exist
func LoadConfigWithDefaults(path string) Config {
	config, _ := LoadConfig(path)
	return config
}
