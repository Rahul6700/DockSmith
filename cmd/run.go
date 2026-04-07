package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"docksmith/state"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [name:tag] [command]",
	Short: "Run a container from a built image",
	// at minimum the user must provide name:tag
	// optionally they can provide a command to override CMD
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// parse name:tag from first arg
		// eg -> myapp:latest gives name=myapp, tag=latest
		parts := strings.SplitN(args[0], ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			fmt.Println("Error: must provide name:tag")
			return
		}
		name, tag := parts[0], parts[1]

		// load all images and find the one matching name:tag
		images, err := state.LoadAllImages()
		if err != nil {
			fmt.Println("Error loading images:", err)
			return
		}

		var img *state.Image
		for i, candidate := range images {
			if candidate.Name == name && candidate.Tag == tag {
				img = &images[i]
				break
			}
		}

		if img == nil {
			fmt.Printf("Error: image %s not found\n", args[0])
			return
		}

		// figure out what command to run
		// if the user passed extra args after name:tag, use those (overrides CMD)
		// otherwise fall back to the CMD stored in the image config
		var command string
		if len(args) > 1 {
			// user provided a command override eg -> docksmith run myapp:latest sh
			command = strings.Join(args[1:], " ")
			fmt.Printf("  [run] overriding CMD with: %s\n", command)
		} else if len(img.Config.Cmd) > 0 {
			// use the CMD baked into the image
			command = strings.Join(img.Config.Cmd, " ")
		} else {
			fmt.Println("Error: no CMD in image and no command provided")
			fmt.Println("Hint: try: docksmith run", args[0], "sh")
			return
		}

		// create a temp dir to use as our rootfs
		// all layers get extracted into here before running the command
		tmpDir, err := os.MkdirTemp("", "docksmith-container-*")
		if err != nil {
			fmt.Println("Error creating container rootfs:", err)
			return
		}
		defer os.RemoveAll(tmpDir)

		fmt.Printf("Starting container from %s:%s\n", name, tag)
		fmt.Printf("  [run] extracting %d layer(s)...\n", len(img.Layers))

		// extract all image layers in order into tmpDir
		// this rebuilds the full filesystem the image was built with
		for _, layer := range img.Layers {
			if err := state.ExtractLayer(layer.Digest, tmpDir); err != nil {
				fmt.Println("Error extracting layer:", err)
				return
			}
		}

		// resolve the working dir inside tmpDir
		// if the image has WorkingDir="/app", we run the command from tmpDir/app
		execDir := tmpDir
		if img.Config.WorkingDir != "" {
			execDir = filepath.Join(tmpDir, img.Config.WorkingDir)
			// create it in case it wasnt part of any layer
			if err := os.MkdirAll(execDir, 0755); err != nil {
				fmt.Println("Error creating workdir:", err)
				return
			}
		}

		fmt.Printf("  [run] workdir: %s\n", img.Config.WorkingDir)
		fmt.Printf("  [run] command: %s\n", command)
		fmt.Printf("  [run] env: %v\n", img.Config.Env)
		fmt.Println("---")

		// build the environment for the container process
		// start with the host env so PATH etc still work
		// then layer the image's ENV vars on top
		env := append(os.Environ(), img.Config.Env...)

		// run the command inside the container rootfs
		c := exec.Command("sh", "-c", command)
		c.Dir = execDir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin // pass stdin through so interactive commands work
		c.Env = env

		// 🔒 same namespace isolation as ExecuteRun during build
		// CLONE_NEWUSER -> unprivileged namespace creation
		// CLONE_NEWUTS  -> own hostname
		// CLONE_NEWPID  -> own PID space
		// CLONE_NEWNS   -> own mount table
		c.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUSER |
				syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWPID |
				syscall.CLONE_NEWNS,
			UidMappings: []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getuid(), Size: 1},
			},
			GidMappings: []syscall.SysProcIDMap{
				{ContainerID: 0, HostID: os.Getgid(), Size: 1},
			},
		}

		if err := c.Run(); err != nil {
			fmt.Println("Container exited with error:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
