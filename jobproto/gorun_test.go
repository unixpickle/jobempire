package jobproto

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGoRun(t *testing.T) {
	master, slave, err := TestingMasterSlave()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()

	tempDir, err := ioutil.TempDir("", "jobproto_test")
	if err != nil {
		t.Fatal(err)
	}
	tempFile := filepath.Join(tempDir, "gorun_out")
	defer func() {
		os.RemoveAll(tempDir)
	}()

	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		for {
			job, err := slave.NextJob()
			if err != nil {
				return
			}
			job.RunTasks(tempDir)
		}
	}()

	job, err := master.StartJob()
	if err != nil {
		t.Fatal(err)
	}
	err = job.Run(&GoRun{
		GoSourceDir: "./test_data/test_go_bin",
		Arguments:   []string{tempFile},
	}, nil)
	if err != nil {
		t.Error("job 1 failed:", err)
	}
	job.Close()
	master.Close()

	select {
	case <-doneChan:
	case <-time.After(time.Second):
		t.Error("slave did not finish before timeout")
	}

	contents, err := ioutil.ReadFile(tempFile)
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(contents, []byte("hello there")) {
		t.Error("unexpected contents:", contents)
	}
}
