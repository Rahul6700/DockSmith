// state.go
package state

import (
	"os"
	"path/filepath"
)

// helper func, used down
// gets the users home dir
// eg -> /home/rahul
func getBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("cannot find home dir")
	}

	// build and return /home/rahul/.docksmith
	return filepath.Join(home, ".docksmith")
}

// checks whether the required dir's exist
// ~/.docksmith/ and all basically
func EnsureStateDirs() error {
	base := getBaseDir()

	// dirs = [/home/rahul/.docksmith/images, /home/rahul/.docksmith/layers, /home/rahul/.docksmith/cache]
	dirs := []string{
		filepath.Join(base, "images"),
		filepath.Join(base, "layers"),
		filepath.Join(base, "cache"),
	}

	for _, dir := range dirs {
		// creating the dir's, if they dont exist
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	
	return nil
}
