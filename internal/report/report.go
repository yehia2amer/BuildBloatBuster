package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"encoding/csv"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/user/BuildBloatBuster/internal/scan"
)

// Reporter handles formatting and displaying scan results
type Reporter struct {
	format string
	sortBy string
}

// NewReporter creates a new reporter with the given format and sort options
func NewReporter(format, sortBy string) *Reporter {
	if format == "" {
		format = "table"
	}
	if sortBy == "" {
		sortBy = "size"
	}
	return &Reporter{
		format: format,
		sortBy: sortBy,
	}
}

// Report displays the candidates according to the configured format
func (r *Reporter) Report(candidates []scan.Candidate, outputDir ...string) error {
	// Sort candidates
	r.sortCandidates(candidates)

	switch r.format {
	case "json":
		return r.reportJSON(candidates)
	case "table":
		return r.reportTable(candidates)
	case "csv":
		if len(outputDir) > 0 {
			return r.reportCSV(candidates, outputDir[0])
		}
		return r.reportCSV(candidates, "")
	default:
		return fmt.Errorf("unsupported format: %s", r.format)
	}
}

func (r *Reporter) reportCSV(candidates []scan.Candidate, outputDir string) error {
	fileName := fmt.Sprintf("BuildBloatBuster-report-%s.csv", time.Now().Format("20060102-150405"))
	var filePath string
	if outputDir == "" {
		filePath = fileName
	} else {
		filePath = filepath.Join(outputDir, fileName)
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV report file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Path", "Size (Bytes)", "Size (Human)", "Reason", "Last Modified"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data
	for _, candidate := range candidates {
		record := []string{
			candidate.Path,
			fmt.Sprintf("%d", candidate.SizeBytes),
			humanize.Bytes(uint64(candidate.SizeBytes)),
			candidate.Reason,
			candidate.NewestMTime.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	fmt.Printf("\nCSV report generated: %s\n", filePath)
	return nil
}

// sortCandidates sorts the candidates based on the configured sort option
func (r *Reporter) sortCandidates(candidates []scan.Candidate) {
	switch r.sortBy {
	case "size":
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].SizeBytes > candidates[j].SizeBytes
		})
	case "path":
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Path < candidates[j].Path
		})
	case "age":
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].NewestMTime.Before(candidates[j].NewestMTime)
		})
	}
}

// reportJSON outputs candidates as JSON
func (r *Reporter) reportJSON(candidates []scan.Candidate) error {
	summary := struct {
		Count      int               `json:"count"`
		TotalSize  int64             `json:"totalSizeBytes"`
		TotalSizeH string            `json:"totalSizeHuman"`
		Candidates []scan.Candidate  `json:"candidates"`
	}{
		Count:      len(candidates),
		TotalSize:  calculateTotalSize(candidates),
		Candidates: candidates,
	}
	summary.TotalSizeH = humanize.Bytes(uint64(summary.TotalSize))

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

// reportTable outputs candidates as a formatted table
func (r *Reporter) reportTable(candidates []scan.Candidate) error {
	if len(candidates) == 0 {
		fmt.Println("No candidates found.")
		return nil
	}

	// Calculate totals
	totalSize := calculateTotalSize(candidates)
	totalCount := len(candidates)

	// Print summary header
	fmt.Printf("Found %d directories using %s\n\n", 
		totalCount, humanize.Bytes(uint64(totalSize)))

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Print table header
	fmt.Fprintln(w, "SIZE\tPATH\tLAST MODIFIED\tREASON")
	fmt.Fprintln(w, "----\t----\t-------------\t------")

	// Print each candidate
	for _, candidate := range candidates {
		sizeStr := humanize.Bytes(uint64(candidate.SizeBytes))
		timeStr := formatTime(candidate.NewestMTime)
		pathStr := truncatePath(candidate.Path, 60)
		reasonStr := truncateString(candidate.Reason, 30)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", 
			sizeStr, pathStr, timeStr, reasonStr)
	}

	// Print summary footer
	fmt.Fprintln(w)
	fmt.Fprintf(w, "TOTAL:\t%s\t%d directories\t\n", 
		humanize.Bytes(uint64(totalSize)), totalCount)

	return nil
}

// calculateTotalSize sums up the size of all candidates
func calculateTotalSize(candidates []scan.Candidate) int64 {
	var total int64
	for _, candidate := range candidates {
		total += candidate.SizeBytes
	}
	return total
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	
	now := time.Now()
	diff := now.Sub(t)
	
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	} else if diff < 30*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	} else {
		return t.Format("2006-01-02")
	}
}

// truncatePath truncates a path to fit within maxLen characters
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	
	// Try to keep the end of the path (filename/dirname)
	if maxLen > 10 {
		return "..." + path[len(path)-(maxLen-3):]
	}
	
	return path[:maxLen-3] + "..."
}

// truncateString truncates a string to fit within maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// PrintScanProgress prints scanning progress information
func PrintScanProgress(scanned, found int) {
	fmt.Printf("\rScanning... %d directories checked, %d candidates found", scanned, found)
}

// PrintSizeProgress prints size calculation progress
func PrintSizeProgress(completed, total int) {
	if total == 0 {
		return
	}
	
	percent := (completed * 100) / total
	bar := strings.Repeat("█", percent/5) + strings.Repeat("░", 20-percent/5)
	fmt.Printf("\rCalculating sizes... [%s] %d%% (%d/%d)", bar, percent, completed, total)
}

// ClearProgress clears the current progress line
func ClearProgress() {
	fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
}