package cmd

import (
	"fmt"
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
				// compute cache key from previous layer + this instruction
				// if the files being copied changed, the layer digest will differ on execution
				// and a new cache entry will be stored
				cacheKey := state.ComputeCacheKey(prevDigest, string(inst.Type), inst.Args)

				if entry := state.GetCacheEntry(cacheKey); entry != nil {
					// cache hit -> skip execution, reuse stored layer
					fmt.Println("  --> cache hit")
					layers = append(layers, entry.Layer)
					prevDigest = entry.Layer.Digest
					continue
				}

				// cache miss -> execute and store result
				layer, err := builder.ExecuteCopy(inst.Args[0], inst.Args[1], contextDir, workDir)
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

