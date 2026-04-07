package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"docksmith/builder"
	"docksmith/state"
	"github.com/spf13/cobra"
)

// build -> creates an image
// run -> makes it executable
// if the image is called myapp:latest
// latest is the tag
var tag string

// hashContextFiles walks the src dir inside contextDir and hashes all file contents
// this is used as part of the COPY cache key so that changing any file busts the cache
func hashContextFiles(contextDir, src string) string {
	srcPath := filepath.Join(contextDir, src)
	h := sha256.New()

	filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		// hash the relative path + file contents
		// so renaming a file also busts the cache
		rel, _ := filepath.Rel(srcPath, path)
		h.Write([]byte(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		h.Write(data)
		return nil
	})

	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

// the build function takes an dir and converts it into an image
// it takes a dir, content hashes it and creates a layer (a .tar file)
// an image is basically layers put together
var buildCmd = &cobra.Command{
	Use:   "build [context]",
	Short: "Build an image from a directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// get the dir which we wanna build
		contextDir := args[0]

		if tag == "" {
			fmt.Println("Error: must provide -t name:tag")
			return
		}

		// parses name and tag
		// if inp was myapp:latest, then name = myapp and tag = latest
		var name, tagVal string
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			fmt.Println("Error: tag must be in format name:tag")
			return
		}
		name, tagVal = parts[0], parts[1]

		// load .docksmithignore patterns from the context dir
		// if the file doesnt exist, patterns will be empty and nothing gets ignored
		patterns, err := builder.LoadIgnorePatterns(contextDir)
		if err != nil {
			fmt.Println("Error reading .docksmithignore:", err)
			return
		}
		if len(patterns) > 0 {
			fmt.Printf("Loaded %d ignore pattern(s) from .docksmithignore\n", len(patterns))
		}

		// parse Docksmithfile
		instructions, err := builder.ParseDocksmithfile("Docksmithfile")
		if err != nil {
			fmt.Println("Parse error:", err)
			return
		}

		var layers []state.Layer

		// track workDir across instructions
		// WORKDIR sets this, and RUN + COPY respect it
		workDir := ""

		// accumulate ENV vars across instructions in KEY=VALUE format
		// each ENV line appends to this slice
		// all of these get passed into every subsequent RUN command
		var envVars []string

		// stores the last CMD seen — last one wins
		var cmdArgs []string

		// prevDigest tracks the digest of the last produced layer
		// it is used as part of the cache key for the next instruction
		// starts empty because there is no previous layer before the first instruction
		prevDigest := ""

		// execute instructions
		for i, inst := range instructions {
			fmt.Printf("Step %d/%d : %s %s\n", i+1, len(instructions), inst.Type, strings.Join(inst.Args, " "))

			switch inst.Type {
			case builder.COPY:
				// hash the actual source files so that changing any file busts the cache
				// without this, the cache key only depends on the instruction args (src, dest)
				// and would never invalidate even if file contents changed
				contentHash := hashContextFiles(contextDir, inst.Args[0])
				cacheKeyArgs := append(inst.Args, contentHash)
				cacheKey := state.ComputeCacheKey(prevDigest, string(inst.Type), cacheKeyArgs)

				if entry := state.GetCacheEntry(cacheKey); entry != nil {
					// cache hit -> skip execution, reuse stored layer
					fmt.Println("  --> cache hit")
					layers = append(layers, entry.Layer)
					prevDigest = entry.Layer.Digest
					continue
				}

				// cache miss -> execute and store result
				// patterns from .docksmithignore are passed in so copyDir can filter files
				layer, err := builder.ExecuteCopy(inst.Args[0], inst.Args[1], contextDir, workDir, patterns)
				if err != nil {
					fmt.Println("COPY failed:", err)
					return
				}
				if err := state.SetCacheEntry(cacheKey, layer); err != nil {
					fmt.Println("Warning: failed to write cache entry:", err)
				}
				layers = append(layers, layer)
				prevDigest = layer.Digest

			case builder.RUN:
				// env vars are part of the cache key too
				// if you add a new ENV before this RUN, the key changes and cache is invalidated
				cacheArgs := append(inst.Args, envVars...)
				cacheKey := state.ComputeCacheKey(prevDigest, string(inst.Type), cacheArgs)

				if entry := state.GetCacheEntry(cacheKey); entry != nil {
					// cache hit -> skip execution, reuse stored layer
					fmt.Println("  --> cache hit")
					layers = append(layers, entry.Layer)
					prevDigest = entry.Layer.Digest
					continue
				}

				// cache miss -> execute and store result
				// we pass envVars so the child process sees all accumulated ENV instructions
				layer, err := builder.ExecuteRun(inst.Args[0], layers, workDir, envVars)
				if err != nil {
					fmt.Println("RUN failed:", err)
					return
				}
				if err := state.SetCacheEntry(cacheKey, layer); err != nil {
					fmt.Println("Warning: failed to write cache entry:", err)
				}
				layers = append(layers, layer)
				prevDigest = layer.Digest

			case builder.WORKDIR:
				workDir = inst.Args[0]
				fmt.Printf("  [WORKDIR] set to %s\n", workDir)

			case builder.ENV:
				// append this KEY=VALUE to our accumulated env list
				// all subsequent RUN commands will see it
				envVars = append(envVars, inst.Args[0])
				fmt.Printf("  [ENV] %s\n", inst.Args[0])

			case builder.CMD:
				// overwrite cmdArgs each time — last CMD wins
				cmdArgs = inst.Args
				fmt.Printf("  [CMD] %s\n", inst.Args[0])
			}
		}

		// creating an obj of the img struct, basically metadata + layers
		img := state.Image{
			Name:    name,
			Tag:     tagVal,
			Digest:  "",
			Created: time.Now().UTC().Format(time.RFC3339),
			Config: state.Config{
				Env:        envVars, // all accumulated ENV vars
				Cmd:        cmdArgs, // last CMD seen (or nil if none)
				WorkingDir: workDir, // populated from WORKDIR instructions
			},
			Layers: layers,
		}

		// computing the images digest (hash basically)
		digest, err := state.ComputeImageDigest(img)
		if err != nil {
			fmt.Println("Error computing digest:", err)
			return
		}

		// setting the digest value in the struct to the one we calculated
		img.Digest = digest

		// write img to disk
		err = state.SaveImage(img)
		if err != nil {
			fmt.Println("Error saving image:", err)
			return
		}

		fmt.Printf("Successfully built %s %s\n", digest, tag)
	},
}

func init() {
	buildCmd.Flags().StringVarP(&tag, "tag", "t", "", "name:tag")
	rootCmd.AddCommand(buildCmd)
}
