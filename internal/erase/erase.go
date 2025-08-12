package erase

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/user/BuildBloatBuster/internal/config"
	"github.com/user/BuildBloatBuster/internal/scan"
)

// Metadata holds information about a quarantined item for restoration.
type Metadata struct {
	OriginalPath  string    `json:"originalPath"`
	QuarantinePath string    `json:"quarantinePath"`
	Timestamp     time.Time `json:"timestamp"`
	SizeBytes     int64     `json:"sizeBytes"`
}

// Eraser handles the deletion of candidates.
type Eraser struct {
	cfg config.Config
}

// NewEraser creates a new Eraser.
func NewEraser(cfg config.Config) *Eraser {
	return &Eraser{cfg: cfg}
}

// EraseCandidates deletes the given candidates based on the configured mode.
func (e *Eraser) EraseCandidates(candidates []scan.Candidate) error {
	switch e.cfg.Delete.Mode {
	case "quarantine":
		return e.quarantineCandidates(candidates)
	case "rm":
		// TODO: Implement permanent deletion
		return fmt.Errorf("permanent deletion mode ('rm') is not yet implemented")
	default:
		return fmt.Errorf("unsupported delete mode: %s", e.cfg.Delete.Mode)
	}
}

// quarantineCandidates moves candidates to the quarantine directory.
func (e *Eraser) quarantineCandidates(candidates []scan.Candidate) error {
	quarantineDir := e.cfg.Delete.QuarantineDir
	if err := os.MkdirAll(quarantineDir, 0755); err != nil {
		return fmt.Errorf("could not create quarantine directory at %s: %w", quarantineDir, err)
	}

	fmt.Printf("Moving %d directories to quarantine (%s)...\n", len(candidates), quarantineDir)

	for _, candidate := range candidates {
		// Create a unique name for the quarantined item
		timestamp := time.Now().Format("20060102-150405")
		baseName := filepath.Base(candidate.Path)
		destName := fmt.Sprintf("%s-%s", timestamp, baseName)
		destPath := filepath.Join(quarantineDir, destName)

		fmt.Printf(" - Quarantining %s -> %s\n", candidate.Path, destPath)

		// Move the directory
		if err := os.Rename(candidate.Path, destPath); err != nil {
			// os.Rename might fail across different devices.
			// A more robust implementation would copy and then delete.
			// For now, we'll just log the error.
			fmt.Fprintf(os.Stderr, "Warning: failed to move %s: %v. It might be on a different device.\n", candidate.Path, err)
			continue // Continue with the next candidate
		}

		// Create metadata file for restoration
		if err := e.writeMetadata(candidate, destPath); err != nil {
			// If metadata fails, we should ideally try to move the directory back.
			// For now, we will log a critical warning.
			fmt.Fprintf(os.Stderr, "CRITICAL: failed to write metadata for %s. Manual restore may be required from %s. Error: %v\n", candidate.Path, destPath, err)
		}
	}

	fmt.Println("\nQuarantine complete.")
	return nil
}

// writeMetadata creates a JSON file with details about the quarantined item.
func (e *Eraser) writeMetadata(candidate scan.Candidate, quarantinePath string) error {
	meta := Metadata{
		OriginalPath:  candidate.Path,
		QuarantinePath: quarantinePath,
		Timestamp:     time.Now(),
		SizeBytes:     candidate.SizeBytes,
	}

	// Metadata file will have the same name as the quarantined dir, but with .json extension
	metaPath := quarantinePath + ".meta.json"

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return os.WriteFile(metaPath, data, 0644)
}