package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"docksmith/state"
)

func ExecuteRun(command string, layers []state.Layer, workDir string, envVars []string) (state.Layer, error) {
	// create temp rootfs
	tmpDir, err := os.MkdirTemp("", "docksmith-run-*")
	if err != nil {
		return state.Layer{}, err
	}
	defer os.RemoveAll(tmpDir)

	// extract all previous layers
	for _, layer := range layers {
		err := state.ExtractLayer(layer.Digest, tmpDir)
		if err != nil {
			return state.Layer{}, fmt.Errorf("failed to extract layer %s: %w", layer.Digest[:12], err)
		}
	}

	// resolve the working dir inside tmpDir
	// if workDir is "/app", the real path on disk is tmpDir + "/app"
	execDir := tmpDir
	if workDir != "" {
		execDir = filepath.Join(tmpDir, workDir)
		// create the workdir if it doesnt exist yet
		if err := os.MkdirAll(execDir, 0755); err != nil {
			return state.Layer{}, fmt.Errorf("failed to create workdir %s: %w", workDir, err)
		}
	}

	fmt.Printf("  [RUN] %s\n", command)
	fmt.Printf("  [RUN] working dir: %s\n", execDir)

	// run command inside tmpDir
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = execDir // assigning the fake dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = nil

	// build the environment for the child process
	// os.Environ() gives us the current host environment (so PATH, HOME etc still work)
	// we then append our accumulated ENV vars on top
	// if the same key exists in both, the later one wins (shell behaviour)
	cmd.Env = append(os.Environ(), envVars...)

	if len(envVars) > 0 {
		fmt.Printf("  [RUN] env vars: %v\n", envVars)
	}

	// 🔒 namespace isolation
	// CLONE_NEWUSER -> own user namespace
	//   this is the key one — linux allows unprivileged processes to create user namespaces
	//   and once inside a user namespace, you are allowed to create all the other namespaces
	// CLONE_NEWUTS  -> own hostname, wont affect the host
	// CLONE_NEWPID  -> own PID space, this process becomes PID 1 inside it
	// CLONE_NEWNS   -> own mount table, mounts wont leak back to the host
	//
	// UidMappings / GidMappings -> tell the kernel how to map UIDs inside the namespace
	//   ContainerID: 0      -> inside the namespace, the process thinks it is root (uid 0)
	//   HostID: os.Getuid() -> but it is actually your current user on the host
	//   Size: 1             -> only map 1 uid (just this one user)
	cmd.SysProcAttr = &syscall.SysProcAttr{
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

	err = cmd.Run()
	if err != nil {
		return state.Layer{}, fmt.Errorf("RUN failed: %w", err)
	}

	// snapshot new state → new layer
	newLayer, err := state.CreateLayerFromDir(tmpDir)
	if err != nil {
		return state.Layer{}, err
	}

	// record which instruction created this layer
	newLayer.CreatedBy = fmt.Sprintf("RUN %s", command)

	return newLayer, nil
}
