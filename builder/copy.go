package builder

import (
	"fmt"
	"os"
	"path/filepath"

	"docksmith/state"
)

// helper func to ExecuteCopy
// copies all the files and sub folders from the source dir to the destination dir
// uses relative file path
// if my file was called /home/rahul/projects/DockSmith/builder/copy.go
// it would read only the relative path (copy.go) and copy that instead of the entire path
// skips any files that match the ignore patterns from .docksmithignore
func copyDir(src, dest string, patterns []string) (included int, ignored int, savedBytes int64, err error) {
	err = filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// always include the root dir itself
		if rel == "." {
			return nil
		}

		// check if this file or dir matches any ignore pattern
		if ShouldIgnore(rel, patterns) {
			ignored++
			savedBytes += info.Size()
			fmt.Printf("  [ignore] %s\n", rel)
			// if its a dir, skip the entire subtree
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		target := filepath.Join(dest, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		// make sure the parent dir exists before writing the file
		// eg -> if target is tmpDir/app/notes.txt, we need tmpDir/app/ to exist first
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		included++
		return os.WriteFile(target, data, 0644)
	})

	return included, ignored, savedBytes, err
}

func ExecuteCopy(src, dest, contextDir, workDir string, patterns []string) (state.Layer, error) {
	// we make a temp dir, which is deleted once the func returns
	tmpDir, err := os.MkdirTemp("", "docksmith-build-*")
	if err != nil {
		return state.Layer{}, err
	}
	defer os.RemoveAll(tmpDir)

	// copy files into temp dir
	// we copy contents of the source dir inside both the tempDir and the destDir
	// tempDir is used to construct the layer
	srcPath := filepath.Join(contextDir, src)
	destPath := filepath.Join(tmpDir, dest)

	included, ignored, savedBytes, err := copyDir(srcPath, destPath, patterns)
	if err != nil {
		return state.Layer{}, err
	}

	// print context stats
	fmt.Printf("  [COPY] files included: %d  ignored: %d  saved: %d bytes\n", included, ignored, savedBytes)

	// create layer from temp dir
	layer, err := state.CreateLayerFromDir(tmpDir)
	if err != nil {
		return state.Layer{}, err
	}

	// record which instruction created this layer
	layer.CreatedBy = fmt.Sprintf("COPY %s %s", src, dest)

	return layer, nil
}
		
