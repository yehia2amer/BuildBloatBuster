package size

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yehia2amer/BuildBloatBuster/internal/scan"
)

func setupSizeTest(t *testing.T) (string, int64, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "size-test-*")
	require.NoError(t, err)

	// Create a file with a known size
	filePath := filepath.Join(tmpDir, "file1.txt")
	fileSize := int64(1024)
	err = os.WriteFile(filePath, make([]byte, fileSize), 0644)
	require.NoError(t, err)

	// Create a subdirectory with another file
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))
	subFilePath := filepath.Join(subDir, "file2.txt")
	subFileSize := int64(2048)
	err = os.WriteFile(subFilePath, make([]byte, subFileSize), 0644)
	require.NoError(t, err)

	totalSize := fileSize + subFileSize

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, totalSize, cleanup
}

func TestCalculator_CalculateSizes(t *testing.T) {
	tmpDir, expectedSize, cleanup := setupSizeTest(t)
	defer cleanup()

	calculator := NewCalculator(4)
	candidates := []scan.Candidate{
		{Path: tmpDir},
	}

	results, err := calculator.CalculateSizes(context.Background(), candidates)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, expectedSize, results[0].SizeBytes)
}

func TestFilterByMinSize(t *testing.T) {
	candidates := []scan.Candidate{
		{SizeBytes: 5 * 1024 * 1024},
		{SizeBytes: 15 * 1024 * 1024},
		{SizeBytes: 25 * 1024 * 1024},
	}

	// Test with a threshold of 10 MB
	filtered := FilterByMinSize(candidates, 10)
	assert.Len(t, filtered, 2)
	assert.Equal(t, int64(15*1024*1024), filtered[0].SizeBytes)
	assert.Equal(t, int64(25*1024*1024), filtered[1].SizeBytes)

	// Test with a threshold of 30 MB
	filtered = FilterByMinSize(candidates, 30)
	assert.Len(t, filtered, 0)

	// Test with no threshold
	filtered = FilterByMinSize(candidates, 0)
	assert.Len(t, filtered, 3)
}
