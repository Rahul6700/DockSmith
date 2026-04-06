// root.go
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

//definf our rootcommand as "docksmith"
// we run this before all our commands -> ex: `docksmith images` to call the images command
var rootCmd = &cobra.Command{
	Use:   "docksmith",
	Short: "A minimal Docker-like system",
}

func Execute() {
	// rootCmd.execute() will parse wtv is there in the CLI after our rootCmd and call that particular execution (example dockersmith images to call imgCmd)
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}


