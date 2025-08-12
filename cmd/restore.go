package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/user/BuildBloatBuster/internal/erase"
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a directory from quarantine",
	Long: `Restores a previously quarantined directory to its original location.
You can run this command without arguments to see a list of restorable items.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRestore()
	},
}

func runRestore() error {
	quarantineDir := Cfg.Delete.QuarantineDir
	items, err := listQuarantinedItems(quarantineDir)
	if err != nil {
		return fmt.Errorf("could not list quarantined items: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("Quarantine is empty. Nothing to restore.")
		return nil
	}

	// Create a list of choices for the prompt
	type promptItem struct {
		erase.Metadata
		HumanSize string
	}

	promptItems := make([]promptItem, len(items))
	for i, item := range items {
		promptItems[i] = promptItem{
			Metadata:  item,
			HumanSize: humanize.Bytes(uint64(item.SizeBytes)),
		}
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "-> {{ .OriginalPath | cyan }} ({{ .HumanSize | red }})",
		Inactive: "   {{ .OriginalPath | faint }} ({{ .HumanSize | faint }})",
		Selected: "Restoring {{ .OriginalPath | green }}",
		Details: `
--------- Item Details ----------
Original Path: {{ .OriginalPath }}
Quarantined At: {{ .Timestamp }}
Size: {{ .HumanSize }}`,
	}

	prompt := promptui.Select{
		Label:     "Select an item to restore",
		Items:     promptItems,
		Templates: templates,
		Size:      10,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrAbort {
			fmt.Println("Restore operation cancelled.")
			return nil
		}
		return fmt.Errorf("prompt failed: %w", err)
	}

	selectedItem := items[idx]

	// Perform the restore
	fmt.Printf("Restoring '%s' to '%s'...\n", selectedItem.QuarantinePath, selectedItem.OriginalPath)
	if err := os.Rename(selectedItem.QuarantinePath, selectedItem.OriginalPath); err != nil {
		return fmt.Errorf("failed to move directory: %w", err)
	}

	// Clean up the metadata file
	metaPath := selectedItem.QuarantinePath + ".meta.json"
	if err := os.Remove(metaPath); err != nil {
		// Log a warning but don't fail the whole operation
		fmt.Fprintf(os.Stderr, "Warning: failed to remove metadata file %s: %v\n", metaPath, err)
	}

	fmt.Println("Restore complete.")
	return nil
}

// listQuarantinedItems scans the quarantine directory for metadata files.
func listQuarantinedItems(quarantineDir string) ([]erase.Metadata, error) {
	var items []erase.Metadata

	files, err := os.ReadDir(quarantineDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Quarantine directory doesn't exist yet
		}
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".meta.json") {
			metaPath := filepath.Join(quarantineDir, file.Name())
			data, err := os.ReadFile(metaPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not read metadata file %s: %v\n", metaPath, err)
				continue
			}

			var meta erase.Metadata
			if err := json.Unmarshal(data, &meta); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not parse metadata file %s: %v\n", metaPath, err)
				continue
			}
			items = append(items, meta)
		}
	}

	return items, nil
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}
