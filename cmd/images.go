package cmd

import (
	"fmt"

	"docksmith/state"
	"github.com/spf13/cobra"
)

// here we define our images command
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List all images",
	// what it actually does
	Run: func(cmd *cobra.Command, args []string) {
		// images is an arr of all the images in Image struct format
		images, err := state.LoadAllImages()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if len(images) == 0 {
			fmt.Println("No images found.")
			return
		}

		// printing all the image details liek docker does
		fmt.Printf("%-10s %-10s %-15s %-25s\n", "NAME", "TAG", "ID", "CREATED")

		for _, img := range images {
			id := img.Digest
			if len(id) > 12 {
				id = id[:12]
			}

			fmt.Printf("%-10s %-10s %-15s %-25s\n",
				img.Name,
				img.Tag,
				id,
				img.Created,
			)
		}
},

}

// adds this as a valid cmd
func init() {
	rootCmd.AddCommand(imagesCmd)
}
