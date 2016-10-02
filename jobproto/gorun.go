package jobproto

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

func init() {
	gob.Register(&GoRun{})
}

// GoRun is a task which runs a Go program on the slave
// by compiling it on the server and transferring the
// executable.
type GoRun struct {
	GoPath      string
	GoSourceDir string
	Arguments   []string
}

// RunMaster runs the master side of the task.
func (g *GoRun) RunMaster(ch TaskChannel) error {
	osArchObj, err := ch.Receive()
	if err != nil {
		return fmt.Errorf("receive platform info: %s", err)
	}
	osArch, ok := osArchObj.([]string)
	if !ok || len(osArch) != 2 {
		return fmt.Errorf("invalid platform info: %v", osArchObj)
	}

	tempDir, err := ioutil.TempDir("", "gorun")
	if err != nil {
		return fmt.Errorf("create temp dir: %s", err)
	}
	defer func() {
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	}()
	tempFile := filepath.Join(tempDir, "executable")

	cmd := exec.Command("go", "build", "-o", tempFile)
	cmd.Env = []string{"GOROOT=" + os.Getenv("GOROOT"), "GOOS=" + osArch[0],
		"GOARCH=" + osArch[1]}
	if g.GoPath != "" {
		cmd.Env = append(cmd.Env, "GOPATH="+g.GoPath)
	} else {
		cmd.Env = append(cmd.Env, "GOPATH="+os.Getenv("GOPATH"))
	}
	cmd.Dir = g.GoSourceDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compile binary: %s", err)
	}

	executable, err := ioutil.ReadFile(tempFile)
	if err != nil {
		return fmt.Errorf("read executable: %s", err)
	}

	os.RemoveAll(tempDir)
	tempDir = ""

	ch.Send(len(executable))
	for i := 0; i < len(executable); i += transferBufferSize {
		if i+transferBufferSize >= len(executable) {
			err = ch.Send(executable[i:])
		} else {
			err = ch.Send(executable[i : i+transferBufferSize])
		}
		if err != nil {
			return fmt.Errorf("send executable: %s", err)
		}
	}

	if err := ch.Send(g.Arguments); err != nil {
		return fmt.Errorf("send arguments: %s", err)
	}

	// Wait for the other end to complete.
	ch.Receive()

	return nil
}

// RunSlave runs the slave side of the task.
func (g *GoRun) RunSlave(root string, ch TaskChannel) error {
	osArch := []string{runtime.GOOS, runtime.GOARCH}
	if err := ch.Send(osArch); err != nil {
		return fmt.Errorf("send platform info: %s", err)
	}

	sizeObj, err := ch.Receive()
	if err != nil {
		return fmt.Errorf("receive size: %s", err)
	}
	size, ok := sizeObj.(int)
	if !ok {
		return fmt.Errorf("bad size type: %T", sizeObj)
	}
	var executable bytes.Buffer
	for executable.Len() < size {
		dataObj, err := ch.Receive()
		if err != nil {
			return fmt.Errorf("receive executable data: %s", err)
		}
		data, ok := dataObj.([]byte)
		if !ok {
			return fmt.Errorf("invalid data type: %T", dataObj)
		}
		executable.Write(data)
	}

	argsObj, err := ch.Receive()
	if err != nil {
		return fmt.Errorf("receive arguments: %s", err)
	}
	args, ok := argsObj.([]string)
	if !ok {
		return fmt.Errorf("invalid argument type: %T", argsObj)
	}

	tempExcPath := filepath.Join(root, fmt.Sprintf("%d", rand.Int63()))
	if err := ioutil.WriteFile(tempExcPath, executable.Bytes(), 0755); err != nil {
		return fmt.Errorf("write executable: %s", err)
	}
	defer os.Remove(tempExcPath)

	var logWg sync.WaitGroup
	cmd := exec.Command(tempExcPath, args...)
	cmd.Dir = root
	if err := logCommandOut(&logWg, cmd, ch); err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start executable: %s", err)
	}

	go func() {
		// If the channel dies because the job or the entire
		// slave session died, we should kill the task.
		ch.Receive()
		cmd.Process.Kill()
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("wait for executable: %s", err)
	}

	logWg.Wait()

	// Notify the other end that we have finished.
	ch.Send(nil)

	return nil
}

func logCommandOut(wg *sync.WaitGroup, cmd *exec.Cmd, ch TaskChannel) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("make stdout pipe: %s", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdout.Close()
		return fmt.Errorf("make stderr pipe: %s", err)
	}
	for i, x := range []io.Reader{stdout, stderr} {
		wg.Add(1)
		go func(name string, r io.Reader) {
			defer wg.Done()
			bufReader := bufio.NewReader(r)
			for {
				line, err := bufReader.ReadString('\n')
				if line != "" || err == nil {
					ch.Log(line)
				}
				if err != nil {
					return
				}
			}
		}([]string{"stdout", "stderr"}[i], x)
	}
	return nil
}
