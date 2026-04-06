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

		// execute instructions
		for i, inst := range instructions {
			fmt.Printf("Step %d/%d : %s %s\n", i+1, len(instructions), inst.Type, strings.Join(inst.Args, " "))

			switch inst.Type {
			case builder.COPY:
				// ExecuteCopy now returns a full Layer struct, not just a digest string
				layer, err := builder.ExecuteCopy(inst.Args[0], inst.Args[1], contextDir, workDir)
				if err != nil {
					fmt.Println("COPY failed:", err)
					return
				}
				layers = append(layers, layer)

			case builder.RUN:
				// ExecuteRun now returns a full Layer struct, not just a digest string
				layer, err := builder.ExecuteRun(inst.Args[0], layers, workDir)
				if err != nil {
					fmt.Println("RUN failed:", err)
					return
				}
				layers = append(layers, layer)

			case builder.WORKDIR:
				workDir = inst.Args[0]
				fmt.Printf("  [WORKDIR] set to %s\n", workDir)
			}
		}

		// creating an obj of the img struct, basically metadata + layers
		img := state.Image{
			Name:    name,
			Tag:     tagVal,
			Digest:  "",
			Created: time.Now().UTC().Format(time.RFC3339),
			Config: state.Config{
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
