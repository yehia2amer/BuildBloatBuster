package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/BuildBloatBuster/internal/report"
	"github.com/user/BuildBloatBuster/internal/scan"
	"github.com/user/BuildBloatBuster/internal/size"
)

var scanCmd = &cobra.Command{
	Use:   "scan [paths...]",
	Short: "Scan for deletable folders",
	Long: `Scans the given paths for folders that can be deleted and shows a report.

By default, scans the current directory for common development artifacts like:
- node_modules (JavaScript/Node.js)
- target (Java/Rust)
- build (C++/Java/Python)
- .cache, .tmp directories
- And many more...

The scan respects your configuration and excludes important directories
like source code, version control folders, and system directories.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runScan(cmd, args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runScan(cmd *cobra.Command, paths []string) error {
	// Override scan paths if provided via command line
	if len(paths) > 0 {
		Cfg.ScanPaths = paths
	}

	if err := checkScanPaths(Cfg.ScanPaths); err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	Cfg.Output.Format = format
	isJSON := Cfg.Output.Format == "json"

	if verbose && !isJSON {
		fmt.Printf("Scanning paths: %v\n", Cfg.ScanPaths)
		fmt.Printf("Include patterns: %v\n", Cfg.IncludeNames)
		fmt.Printf("Min size: %d MB\n", Cfg.MinSizeMB)
		fmt.Printf("Max depth: %d\n", Cfg.MaxDepth)
		fmt.Printf("Concurrency: %d\n", Cfg.Concurrency)
		fmt.Println()
	}

	// Create scanner
	scanner := scan.NewScanner(Cfg)

	// Start scanning
	if verbose && !isJSON {
		fmt.Println("Scanning directories...")
	}

	startTime := time.Now()
	candidates, err := scanner.ScanPaths()
	if err != nil {
		return fmt.Errorf("scanning failed: %w", err)
	}

	if verbose && !isJSON {
		fmt.Printf("Found %d candidates in %v\n", len(candidates), time.Since(startTime))
	}

	if len(candidates) == 0 {
		if !isJSON {
			fmt.Println("No directories found matching the criteria.")
		}
		return nil
	}

	// Calculate sizes concurrently
	if verbose && !isJSON {
		fmt.Println("Calculating sizes...")
	}

	calculator := size.NewCalculator(Cfg.Concurrency)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startTime = time.Now()
	candidates, err = calculator.CalculateSizes(ctx, candidates)
	if err != nil {
		return fmt.Errorf("size calculation failed: %w", err)
	}

	if verbose && !isJSON {
		fmt.Printf("Size calculation completed in %v\n", time.Since(startTime))
	}

	// Filter by minimum size
	candidates = size.FilterByMinSize(candidates, Cfg.MinSizeMB)

	if len(candidates) == 0 {
		if !isJSON {
			fmt.Printf("No directories found larger than %d MB.\n", Cfg.MinSizeMB)
		}
		return nil
	}

	// Generate report
	reporter := report.NewReporter(Cfg.Output.Format, Cfg.Output.SortBy)
	return reporter.Report(candidates)
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Add scan-specific flags
	scanCmd.Flags().IntP("min-size", "s", 0, "minimum size in MB (overrides config)")
	scanCmd.Flags().IntP("max-depth", "d", 0, "maximum directory depth (overrides config)")
	scanCmd.Flags().StringSliceP("include", "i", nil, "additional patterns to include")
	scanCmd.Flags().StringSliceP("exclude", "e", nil, "additional patterns to exclude")
	scanCmd.Flags().String("format", "table", "output format (table, json, csv)")
}
