package main

import (
	"fmt"
	"os"
	// other packages
	"docksmith/state"
)

func main() {

	// this is func to create ~/.docksmith and the images, layers and chche dirs in it
	err := state.EnsureStateDirs()
	if err != nil {
		fmt.Println("Error initializing state:", err)
		os.Exit(1)
	}

}
