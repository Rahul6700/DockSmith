package cmd

import (
	"fmt"
	"time"
	"strings"
	"github.com/spf13/cobra"
	"docksmith/state"
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
		// n, err := fmt.Sscanf(tag, "%[^:]:%s", &name, &tagVal)
		// if err != nil || n != 2 {
		// 	fmt.Println("Error: tag must be in format name:tag")
		// 	return
		// }

		parts := strings.SplitN(tag, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			fmt.Println("Error: tag must be in format name:tag")
			return
		}
		name, tagVal = parts[0], parts[1]

		fmt.Println("Step 1/1 : Creating layer from context")

		// func from layer.go, takes in the dir and creates an returns a layer
		layer, err := state.CreateLayerFromDir(contextDir)
		if err != nil {
			fmt.Println("Error creating layer:", err)
			return
		}
		
		// creating an an obj of the img struct, basically metadata + layers
		img := state.Image{
			Name:    name,
			Tag:     tagVal,
			Digest:  "",
			Created: time.Now().UTC().Format(time.RFC3339),
			Config:  state.Config{},
			Layers:  []state.Layer{layer},
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
