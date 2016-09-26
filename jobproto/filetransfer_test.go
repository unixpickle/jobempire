package jobproto

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileTransfer(t *testing.T) {
	master, slave, err := TestingMasterSlave()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()

	tempDir, err := ioutil.TempDir("", "jobproto_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.RemoveAll(tempDir)
	}()

	sourceFile := filepath.Join(tempDir, "source_file")
	sourceData := []byte("hello world")
	if err := ioutil.WriteFile(sourceFile, sourceData, 0755); err != nil {
		t.Fatal(err)
	}

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
	err = job.Run(&FileTransfer{
		ToSlave:    true,
		SlavePath:  "dest_file",
		MasterPath: sourceFile,
	}, nil)
	if err != nil {
		t.Error("job 1 failed:", err)
	}
	err = job.Run(&FileTransfer{
		ToSlave:    false,
		SlavePath:  "source_file",
		MasterPath: filepath.Join(tempDir, "dest_file1"),
	}, nil)
	if err != nil {
		t.Error("job 2 failed:", err)
	}
	job.Close()
	master.Close()

	select {
	case <-doneChan:
	case <-time.After(time.Second):
		t.Error("slave did not finish before timeout")
	}

	for _, subPath := range []string{"dest_file", "dest_file1", "source_file"} {
		path := filepath.Join(tempDir, subPath)
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %s", subPath, err)
		} else {
			if !bytes.Equal(contents, sourceData) {
				t.Errorf("bad contents for: %s", subPath)
			}
		}
	}
}
