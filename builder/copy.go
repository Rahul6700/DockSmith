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
func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func ExecuteCopy(src, dest, contextDir, workDir string) (state.Layer, error) {
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
	err = copyDir(srcPath, destPath)
	if err != nil {
		return state.Layer{}, err
	}

	// create layer from temp dir
	layer, err := state.CreateLayerFromDir(tmpDir)
	if err != nil {
		return state.Layer{}, err
	}

	// record which instruction created this layer
	layer.CreatedBy = fmt.Sprintf("COPY %s %s", src, dest)

	return layer, nil
}
