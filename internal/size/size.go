package size

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"golang.org/x/sync/errgroup"

	"github.com/yehia2amer/BuildBloatBuster/internal/scan"
)

// Calculator handles concurrent size calculation for directories
type Calculator struct {
	concurrency int
}

// NewCalculator creates a new size calculator
func NewCalculator(concurrency int) *Calculator {
	if concurrency <= 0 {
		concurrency = 8 // Default fallback
	}
	return &Calculator{
		concurrency: concurrency,
	}
}

// CalculateSizes calculates sizes for all candidates concurrently
func (c *Calculator) CalculateSizes(ctx context.Context, candidates []scan.Candidate) ([]scan.Candidate, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}

	// Create channels for work distribution
	jobs := make(chan int, len(candidates))
	results := make([]scan.Candidate, len(candidates))

	// Use errgroup for proper error handling and cancellation
	g, ctx := errgroup.WithContext(ctx)

	// Initialize progress bar
	p := mpb.New(mpb.WithWidth(60), mpb.WithRefreshRate(180*time.Millisecond))
	bar := p.New(int64(len(candidates)),
		mpb.BarStyle().Lbound("[").Filler("=").Tip(">").Padding("-").Rbound("]"),
		mpb.PrependDecorators(
			decor.Name("Calculating sizes "),
			decor.CountersNoUnit("%d / %d"),
		),
		mpb.AppendDecorators(
			decor.Percentage(),
			decor.Name(" | "),
			decor.Elapsed(decor.ET_STYLE_GO),
		),
	)

	// Start workers
	for i := 0; i < c.concurrency; i++ {
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case idx, ok := <-jobs:
					if !ok {
						return nil // Channel closed, worker done
					}

					// Calculate size for this candidate
					size, err := c.calculateDirectorySize(candidates[idx].Path)
					if err != nil {
						// Log error but don't fail the whole operation
						// Note: In a real app, this should go to a proper logger
						// and not interfere with the progress bar rendering.
					}

					// Update result
					results[idx] = candidates[idx]
					results[idx].SizeBytes = size

					// Increment progress bar
					bar.Increment()
				}
			}
		})
	}

	// Send jobs to workers
	go func() {
		defer close(jobs)
		for i := range candidates {
			select {
			case <-ctx.Done():
				return
			case jobs <- i:
			}
		}
	}()

	// Wait for all workers to complete
	err := g.Wait()

	// Wait for the progress bar to finish
	p.Wait()

	if err != nil {
		return nil, err
	}

	return results, nil
}

// calculateDirectorySize calculates the total size of a directory
func (c *Calculator) calculateDirectorySize(dirPath string) (int64, error) {
	var totalSize int64
	var mutex sync.Mutex

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip files/directories we can't access
			if os.IsPermission(err) || os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return nil // Skip files we can't stat
			}

			mutex.Lock()
			totalSize += info.Size()
			mutex.Unlock()
		}

		return nil
	})

	return totalSize, err
}

// CalculateDirectorySize is a convenience function for calculating a single directory size
func CalculateDirectorySize(dirPath string) (int64, error) {
	calc := NewCalculator(1)
	return calc.calculateDirectorySize(dirPath)
}

// FilterByMinSize filters candidates by minimum size threshold
func FilterByMinSize(candidates []scan.Candidate, minSizeMB int) []scan.Candidate {
	if minSizeMB <= 0 {
		return candidates
	}

	minSizeBytes := int64(minSizeMB) * 1024 * 1024
	var filtered []scan.Candidate

	for _, candidate := range candidates {
		if candidate.SizeBytes >= minSizeBytes {
			filtered = append(filtered, candidate)
		}
	}

	return filtered
}
