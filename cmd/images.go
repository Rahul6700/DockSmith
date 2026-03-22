package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// here we define our images command
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List all images",
	// what it actually does
	Run: func(cmd *cobra.Command, args []string) {
		// todo this part
		// for now print placeholder saying no images
		fmt.Println("No images yet.")
	},
}


// adds this as a valid cmd
func init() {
	rootCmd.AddCommand(imagesCmd)
}
