package cmd
import (
	"fmt"
	"time"
	"strings"
	"github.com/spf13/cobra"
	"docksmith/state"
	"docksmith/builder"
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

		// parse Docksmithfile
		instructions, err := builder.ParseDocksmithfile("Docksmithfile")
		if err != nil {
			fmt.Println("Parse error:", err)
			return
		}

		var layers []state.Layer

		// execute instructions
		for i, inst := range instructions {
			fmt.Printf("Step %d/%d : %s\n", i+1, len(instructions), inst.Type)

			switch inst.Type {

			case builder.COPY:
				digest, err := builder.ExecuteCopy(inst.Args[0], inst.Args[1], contextDir, "")
				if err != nil {
					fmt.Println("COPY failed:", err)
					return
				}

				layers = append(layers, state.Layer{
					Digest: digest,
				})
			}
		}
		
		// creating an an obj of the img struct, basically metadata + layers
		img := state.Image{
			Name:    name,
			Tag:     tagVal,
			Digest:  "",
			Created: time.Now().UTC().Format(time.RFC3339),
			Config:  state.Config{},
			Layers:  layers,
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
