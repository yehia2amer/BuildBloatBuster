package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Permanently delete items from quarantine",
	Long: `Permanently deletes items from the quarantine directory.
Use the --days flag to only purge items older than a certain number of days.
WARNING: This action is irreversible.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		days, _ := cmd.Flags().GetInt("days")
		return runPurge(days)
	},
}

func runPurge(days int) error {
	quarantineDir := Cfg.Delete.QuarantineDir
	items, err := listQuarantinedItems(quarantineDir)
	if err != nil {
		return fmt.Errorf("could not list quarantined items: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("Quarantine is empty. Nothing to purge.")
		return nil
	}

	var toPurge []string
	var toPurgeMeta []string
	var cutoff time.Time
	if days > 0 {
		cutoff = time.Now().AddDate(0, 0, -days)
	}

	for _, item := range items {
		if days == 0 || item.Timestamp.Before(cutoff) {
			toPurge = append(toPurge, item.QuarantinePath)
			toPurgeMeta = append(toPurgeMeta, item.QuarantinePath+".meta.json")
		}
	}

	if len(toPurge) == 0 {
		fmt.Printf("No items found in quarantine older than %d days.\n", days)
		return nil
	}

	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("Permanently delete %d items from quarantine? This cannot be undone.", len(toPurge)),
		IsConfirm: true,
		Default:   "n",
	}
	_, err = prompt.Run()
	if err != nil {
		if err == promptui.ErrAbort {
			fmt.Println("Purge operation cancelled.")
			return nil
		}
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Perform purge
	fmt.Println("Purging items...")
	for i, path := range toPurge {
		fmt.Printf(" - Deleting %s\n", path)
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete directory %s: %v\n", path, err)
		}
		// Also delete metadata file
		if err := os.Remove(toPurgeMeta[i]); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete metadata file %s: %v\n", toPurgeMeta[i], err)
		}
	}

	fmt.Println("Purge complete.")
	return nil
}

func init() {
	rootCmd.AddCommand(purgeCmd)
	purgeCmd.Flags().Int("days", 0, "only purge items older than this many days (default: all items)")
}
