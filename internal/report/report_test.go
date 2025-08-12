package report

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/BuildBloatBuster/internal/scan"
)

func TestReporter_JSON(t *testing.T) {
	candidates := []scan.Candidate{
		{Path: "/tmp/project/node_modules", SizeBytes: 200000000, Reason: "node_modules", NewestMTime: time.Now()},
		{Path: "/tmp/project/target", SizeBytes: 50000000, Reason: "target", NewestMTime: time.Now().Add(-24 * time.Hour)},
	}

	reporter := NewReporter("json", "size")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(candidates)
	require.NoError(t, err)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Unmarshal and verify
	var summary struct {
		Count      int              `json:"count"`
		TotalSize  int64            `json:"totalSizeBytes"`
		Candidates []scan.Candidate `json:"candidates"`
	}

	err = json.Unmarshal(buf.Bytes(), &summary)
	require.NoError(t, err, "output should be valid JSON")

	assert.Equal(t, 2, summary.Count)
	assert.Equal(t, int64(250000000), summary.TotalSize)
	assert.Len(t, summary.Candidates, 2)
	assert.Equal(t, "/tmp/project/node_modules", summary.Candidates[0].Path)
}

func TestReporter_CSV(t *testing.T) {
	candidates := []scan.Candidate{
		{Path: "/tmp/project/node_modules", SizeBytes: 200000000, Reason: "node_modules", NewestMTime: time.Now()},
		{Path: "/tmp/project/target", SizeBytes: 50000000, Reason: "target", NewestMTime: time.Now().Add(-24 * time.Hour)},
	}

	reporter := NewReporter("csv", "size")

	// For this test, we'll just check that it runs without error
	// and creates a file. A more robust test would parse the CSV.
	tmpDir, err := os.MkdirTemp("", "report-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = reporter.Report(candidates, tmpDir)
	require.NoError(t, err)

	// Find the created report file
	matches, err := filepath.Glob(filepath.Join(tmpDir, "BuildBloatBuster-report-*.csv"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "CSV report file should have been created")
}