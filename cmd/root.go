package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yehia2amer/BuildBloatBuster/internal/config"
)

var cfgFile string
var Cfg config.Config
var version string

// Global flags
var (
	dryRun     bool
	jsonOutput bool
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "BuildBloatBuster",
	Short: "A CLI tool to clean up development folders",
	Long: `BuildBloatBuster is a fast and safe CLI tool to remove auto-generated development folders
like node_modules, target, build, .cache and other common build artifacts.

It operates with safety as the primary concern:
- Dry-run mode by default
- Quarantine deletion (move to trash) instead of permanent deletion
- Smart filtering to avoid deleting important directories
- Interactive confirmation prompts`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load configuration
		if cfgFile != "" {
			var err error
			Cfg, err = config.LoadConfig(cfgFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config file %s: %v\n", cfgFile, err)
				os.Exit(1)
			}
			if verbose {
				fmt.Printf("Using config file: %s\n", cfgFile)
			}
		} else {
			// Try to load from default locations
			Cfg = config.LoadConfigWithDefaults(".BuildBloatBuster.yaml")
			if verbose {
				fmt.Println("Using configuration with defaults")
			}
		}

	},
}

func Execute() {
	startTime := time.Now()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nTotal time taken: %v\n", time.Since(startTime))
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./.BuildBloatBuster.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", true, "show what would be deleted without actually deleting")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output results in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.Version = version
}

// SetVersionInfo sets the version information for the CLI
func SetVersionInfo(v, commit, date string) {
	version = fmt.Sprintf("%s (commit: %s, built at: %s)", v, commit, date)
}
