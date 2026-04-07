package builder

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadIgnorePatterns reads a .docksmithignore file and returns the list of patterns
// if the file doesnt exist, it returns an empty slice (no patterns = ignore nothing)
// lines starting with # are comments and are skipped
// blank lines are skipped too
func LoadIgnorePatterns(contextDir string) ([]string, error) {
	path := filepath.Join(contextDir, ".docksmithignore")

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// no .docksmithignore file -> thats fine, just ignore nothing
			return []string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		patterns = append(patterns, line)
	}

	return patterns, nil
}

// ShouldIgnore checks whether a given file path matches any of the ignore patterns
// path is the relative path of the file from the context dir eg -> "node_modules/lodash/index.js"
// patterns are the lines from .docksmithignore eg -> ["node_modules", "*.log", ".git"]
func ShouldIgnore(relPath string, patterns []string) bool {
	// get just the file or dir name eg -> "index.js" from "node_modules/lodash/index.js"
	base := filepath.Base(relPath)

	for _, pattern := range patterns {
		// strip trailing slash from patterns like "build/" -> "build"
		// we handle dir matching by checking both the base name and the full path
		cleanPattern := strings.TrimSuffix(pattern, "/")

		// check against the base name first eg -> "*.log" matches "error.log"
		if matched, _ := filepath.Match(cleanPattern, base); matched {
			return true
		}

		// check against the full relative path eg -> "build/output" matches "build/output/file.txt"
		if matched, _ := filepath.Match(cleanPattern, relPath); matched {
			return true
		}

		// check if any component of the path matches eg -> "node_modules" should ignore
		// "node_modules/lodash/index.js" by matching the first segment
		parts := strings.Split(relPath, string(filepath.Separator))
		for _, part := range parts {
			if matched, _ := filepath.Match(cleanPattern, part); matched {
				return true
			}
		}
	}

	return false
}
