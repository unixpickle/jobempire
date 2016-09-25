package jobproto

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func init() {
	gob.Register(&GoRun{})
}

// GoRun is a task which runs a Go program on the slave
// by compiling it on the server and transferring the
// executable.
type GoRun struct {
	GoSourceDir string
	Arguments   []string
}

// RunMaster runs the master side of the task.
func (g *GoRun) RunMaster(ch TaskChannel) error {
	osArchObj, err := ch.Receive()
	if err != nil {
		return fmt.Errorf("failed to receive platform info: %s", err)
	}
	osArch, ok := osArchObj.([]string)
	if !ok || len(osArch) != 2 {
		return fmt.Errorf("unexpected platform info: %v", osArchObj)
	}

	tempDir, err := ioutil.TempDir("", "gorun")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %s", err)
	}
	defer func() {
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	}()
	tempFile := filepath.Join(tempDir, "executable")

	cmd := exec.Command("go", "build", "-o", tempFile, g.GoSourceDir)
	cmd.Env = []string{"GOPATH", os.Getenv("GOPATH"), "GOOS", osArch[0], "GOARCH", osArch[1]}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to compile binary: %s", err)
	}

	executable, err := ioutil.ReadFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to read executable: %s", err)
	}

	os.RemoveAll(tempDir)
	tempDir = ""

	if err := ch.Send(executable); err != nil {
		return fmt.Errorf("failed to send executable: %s", err)
	}
	if err := ch.Send(g.Arguments); err != nil {
		return fmt.Errorf("failed to send arguments: %s", err)
	}

	// Wait for the other end to complete.
	ch.Receive()

	return nil
}

// RunSlave runs the slave side of the task.
func (g *GoRun) RunSlave(root string, ch TaskChannel) error {
	osArch := []string{runtime.GOOS, runtime.GOARCH}
	if err := ch.Send(osArch); err != nil {
		return fmt.Errorf("failed to send platform info: %s", err)
	}

	executableObj, err := ch.Receive()
	if err != nil {
		return fmt.Errorf("failed to receive executable: %s", err)
	}
	executable, ok := executableObj.([]byte)
	if !ok {
		return fmt.Errorf("unexpected type for executable: %T", executableObj)
	}

	argsObj, err := ch.Receive()
	if err != nil {
		return fmt.Errorf("failed to receive arguments: %s", err)
	}
	args, ok := argsObj.([]string)
	if !ok {
		return fmt.Errorf("unexpected type for arguments: %T", argsObj)
	}

	tempExcPath := filepath.Join(root, fmt.Sprintf("%d", rand.Int63()))
	if err := ioutil.WriteFile(tempExcPath, executable, 0755); err != nil {
		return fmt.Errorf("failed to write executable: %s", err)
	}
	defer os.Remove(tempExcPath)

	cmd := exec.Command(tempExcPath, args...)
	cmd.Dir = root
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting executable: %s", err)
	}

	go func() {
		// If the channel dies because the job or the entire
		// slave session died, we should kill the task.
		ch.Receive()
		cmd.Process.Kill()
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error from executable: %s", err)
	}

	// Notify the other end that we have finished.
	ch.Send(nil)

	return nil
}
