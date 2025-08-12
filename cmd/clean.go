package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/yehia2amer/BuildBloatBuster/internal/erase"
	"github.com/yehia2amer/BuildBloatBuster/internal/report"
	"github.com/yehia2amer/BuildBloatBuster/internal/scan"
	"github.com/yehia2amer/BuildBloatBuster/internal/size"
)

var cleanCmd = &cobra.Command{
	Use:   "clean [paths...]",
	Short: "Clean up deletable folders",
	Long:  `Scans for and deletes specified folders, with a confirmation prompt.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runClean(cmd, args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runClean(cmd *cobra.Command, paths []string) error {
	if err := checkScanPaths(Cfg.ScanPaths); err != nil {
		return err
	}
	// This function is a modified version of runScan to allow for interaction.
	// 1. Scan for candidates
	format, _ := cmd.Flags().GetString("format")
	Cfg.Output.Format = format
	candidates, err := findCandidates(paths)
	if err != nil {
		return err
	}

	if len(candidates) == 0 {
		fmt.Println("No directories found to clean.")
		return nil
	}

	isJSON := Cfg.Output.Format == "json"

	// 2. Report candidates to the user
	reporter := report.NewReporter(Cfg.Output.Format, Cfg.Output.SortBy)
	if err := reporter.Report(candidates); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// 3. Handle dry-run or prompt for confirmation
	if dryRun {
		if !isJSON {
			fmt.Println("\nDry run enabled. No files will be deleted.")
			fmt.Println("Run with --dry-run=false to enable deletion.")
		}
		return nil
	}

	// If not a dry run, prompt for confirmation unless --yes is passed or in JSON mode
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes && !isJSON {
		proceed, err := confirmDeletion(candidates)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !proceed {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// 4. Perform deletion
	eraser := erase.NewEraser(Cfg)
	if err := eraser.EraseCandidates(candidates); err != nil {
		return fmt.Errorf("failed during deletion: %w", err)
	}

	return nil
}

// findCandidates performs the scan and size calculation, returning the final list.
func findCandidates(paths []string) ([]scan.Candidate, error) {
	if len(paths) > 0 {
		Cfg.ScanPaths = paths
	}

	scanner := scan.NewScanner(Cfg)
	candidates, err := scanner.ScanPaths()
	if err != nil {
		return nil, fmt.Errorf("scanning failed: %w", err)
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	calculator := size.NewCalculator(Cfg.Concurrency)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	candidates, err = calculator.CalculateSizes(ctx, candidates)
	if err != nil {
		return nil, fmt.Errorf("size calculation failed: %w", err)
	}

	return size.FilterByMinSize(candidates, Cfg.MinSizeMB), nil
}

func confirmDeletion(candidates []scan.Candidate) (bool, error) {
	var totalSize int64
	for _, c := range candidates {
		totalSize += c.SizeBytes
	}
	totalSizeStr := humanize.Bytes(uint64(totalSize))
	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("Delete %d directories and free %s of space?", len(candidates), totalSizeStr),
		IsConfirm: true,
		Default:   "n",
	}

	_, err := prompt.Run()

	if err != nil {
		if err == promptui.ErrAbort {
			return false, nil // User cancelled
		}
		return false, err // Other error
	}

	return true, nil // User confirmed
}

func init() {
	rootCmd.AddCommand(cleanCmd)

	// Add flags from scan command to clean command
	cleanCmd.Flags().IntP("min-size", "s", 0, "minimum size in MB (overrides config)")
	cleanCmd.Flags().IntP("max-depth", "d", 0, "maximum directory depth (overrides config)")
	cleanCmd.Flags().StringSliceP("include", "i", nil, "additional patterns to include")
	cleanCmd.Flags().StringSliceP("exclude", "e", nil, "additional patterns to exclude")
	cleanCmd.Flags().BoolP("yes", "y", false, "skip confirmation prompt and proceed with deletion")
	cleanCmd.Flags().String("format", "table", "output format (table, json, csv)")
}
