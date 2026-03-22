package main

import (
	"fmt"
	"os"
	// other packages
	"docksmith/state"
	"docksmith/cmd"
)

func main() {

	// this is func to create ~/.docksmith and the images, layers and chche dirs in it
	err := state.EnsureStateDirs()
	if err != nil {
		fmt.Println("Error initializing state:", err)
		os.Exit(1)
	}

	cmd.Execute()

// 	img := state.Image{
// 	Name:    "myapp",
// 	Tag:     "latest",
// 	Digest:  "sha256:dummyhash123456789",
// 	Created: "2026-03-22T10:00:00Z",
// 	Config:  state.Config{},
// 	Layers:  []state.Layer{},
// }
//
// state.SaveImage(img)

}
