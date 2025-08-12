package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/BuildBloatBuster/internal/config"
)

// Candidate represents a directory that can be deleted
type Candidate struct {
	Path        string    `json:"path"`
	SizeBytes   int64     `json:"sizeBytes"`
	Reason      string    `json:"reason"`
	NewestMTime time.Time `json:"newestMTime"`
}

// Scanner handles directory scanning operations
type Scanner struct {
	config      config.Config
	includeMap  map[string]struct{}
	excludeMap  map[string]struct{}
	excludePaths map[string]struct{}
}

// NewScanner creates a new scanner with the given configuration
func NewScanner(cfg config.Config) *Scanner {
	s := &Scanner{
		config:       cfg,
		includeMap:   make(map[string]struct{}),
		excludeMap:   make(map[string]struct{}),
		excludePaths: make(map[string]struct{}),
	}

	// Build lookup maps for O(1) access
	for _, name := range cfg.IncludeNames {
		s.includeMap[name] = struct{}{}
	}
	for _, name := range cfg.ExcludeNames {
		s.excludeMap[name] = struct{}{}
	}
	for _, path := range cfg.ExcludePaths {
		absPath, err := filepath.Abs(path)
		if err == nil {
			s.excludePaths[absPath] = struct{}{}
		}
		s.excludePaths[path] = struct{}{} // Also store original path
	}

	return s
}

// ScanPaths scans all configured paths and returns candidates
func (s *Scanner) ScanPaths() ([]Candidate, error) {
	var allCandidates []Candidate

	for _, scanPath := range s.config.ScanPaths {
		candidates, err := s.scanPath(scanPath)
		if err != nil {
			return nil, fmt.Errorf("error scanning path %s: %w", scanPath, err)
		}
		allCandidates = append(allCandidates, candidates...)
	}

	return allCandidates, nil
}

// scanPath scans a single path for candidates
func (s *Scanner) scanPath(rootPath string) ([]Candidate, error) {
	var candidates []Candidate

	absRootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get absolute path for %s: %w", rootPath, err)
	}

	// Check if root path itself is excluded
	if s.isPathExcluded(absRootPath) {
		return candidates, nil // Skip entirely
	}

	err = filepath.WalkDir(absRootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't read
			if os.IsPermission(err) {
				return filepath.SkipDir
			}
			return err
		}

		if !d.IsDir() {
			return nil // Skip files
		}

		// Get relative depth from root
		relPath, err := filepath.Rel(absRootPath, path)
		if err != nil {
			return nil
		}

		depth := strings.Count(relPath, string(filepath.Separator))
		if relPath == "." {
			depth = 0
		}

		// Check max depth
		if s.config.MaxDepth > 0 && depth >= s.config.MaxDepth {
			return filepath.SkipDir
		}

		// Check if path is excluded
		if s.isPathExcluded(path) {
			return filepath.SkipDir
		}

		// Check if this is a symlink and we're not following them
		if !s.config.FollowSymlinks {
			if info, err := d.Info(); err == nil && info.Mode()&os.ModeSymlink != 0 {
				return filepath.SkipDir
			}
		}

		dirName := d.Name()

		// Check if directory name is a VCS dir
		if s.isVersionControlDir(dirName) {
			return filepath.SkipDir
		}

		// Check if directory name is excluded
		if _, excluded := s.excludeMap[dirName]; excluded {
			return filepath.SkipDir
		}

		// Check if directory name is included
		if _, included := s.includeMap[dirName]; included {
			// This is a candidate, don't descend into it
			candidate := Candidate{
				Path:      path,
				Reason:    fmt.Sprintf("matches include pattern '%s'", dirName),
				SizeBytes: 0, // Will be calculated later
			}

			// Get modification time
			if info, err := d.Info(); err == nil {
				candidate.NewestMTime = info.ModTime()
			}

			candidates = append(candidates, candidate)
			return filepath.SkipDir
		}

		// Continue traversing
		return nil
	})

	if err != nil {
		return nil, err
	}

	return candidates, nil
}

// isPathExcluded checks if a path should be excluded
func (s *Scanner) isPathExcluded(path string) bool {
	// Check direct path exclusion
	if _, excluded := s.excludePaths[path]; excluded {
		return true
	}

	// Check if path is under any excluded directory
	for excludePath := range s.excludePaths {
		if strings.HasPrefix(path, excludePath+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

// isVersionControlDir checks if the directory name is a version control directory.
func (s *Scanner) isVersionControlDir(dirName string) bool {
	switch dirName {
	case ".git", ".svn", ".hg", ".bzr":
		return true
	default:
		return false
	}
}

// IsSafeToDelete performs additional safety checks on a candidate
func (s *Scanner) IsSafeToDelete(candidate Candidate) bool {
	// Don't delete if it's a version control directory
	if s.isVersionControlDir(candidate.Path) {
		return false
	}

	// Don't delete if it's an excluded path
	if s.isPathExcluded(candidate.Path) {
		return false
	}

	// Don't delete if it's under a project root (contains .git, package.json, etc.)
	// unless it's in our safe include list
	parentDir := filepath.Dir(candidate.Path)
	if s.isProjectRoot(parentDir) {
		// Only allow if the directory name is in our include list
		dirName := filepath.Base(candidate.Path)
		_, included := s.includeMap[dirName]
		return included
	}

	return true
}

// isProjectRoot checks if a directory appears to be a project root
func (s *Scanner) isProjectRoot(path string) bool {
	projectFiles := []string{
		".git", ".svn", ".hg",
		"package.json", "package-lock.json", "yarn.lock",
		"go.mod", "go.sum",
		"Cargo.toml", "Cargo.lock",
		"pom.xml", "build.gradle", "build.gradle.kts",
		"requirements.txt", "setup.py", "pyproject.toml",
		"Gemfile", "Gemfile.lock",
		"composer.json", "composer.lock",
	}

	for _, file := range projectFiles {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			return true
		}
	}

	return false
}