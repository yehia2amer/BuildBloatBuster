package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/user/BuildBloatBuster/internal/config"
)

func checkScanPaths(scanPaths []string) error {
	protectedPaths := config.GetProtectedPaths()
	for _, scanPath := range scanPaths {
		absScanPath, err := filepath.Abs(scanPath)
		if err != nil {
			// If we can't get an absolute path, play it safe
			return fmt.Errorf("could not verify safety of path %s: %w", scanPath, err)
		}

		for _, protected := range protectedPaths {
			absProtected, err := filepath.Abs(protected)
			if err != nil {
				continue
			}
			if absScanPath == absProtected {
				return fmt.Errorf("for your safety, scanning protected path '%s' is not allowed", scanPath)
			}
		}
	}
	return nil
}
