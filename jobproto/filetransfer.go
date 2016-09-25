package jobproto

import (
	"encoding/gob"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
)

func init() {
	gob.Register(&FileTransfer{})
}

// FileTransfer is a Task that implements a file transfer
// between a master and a slave.
type FileTransfer struct {
	// If ToSlave is true, the file is being uploaded to
	// the slave.
	ToSlave bool

	MasterPath string
	SlavePath  string
}

// RunMaster runs the master's end of the file transfer.
func (f *FileTransfer) RunMaster(ch TaskChannel) error {
	if f.ToSlave {
		return f.runSender(f.MasterPath, ch)
	} else {
		return f.runReceiver(f.MasterPath, ch)
	}
}

// RunSlave runs the slave's end of the file transfer.
func (f *FileTransfer) RunSlave(root string, ch TaskChannel) error {
	path := filepath.Join(root, f.SlavePath)
	if f.ToSlave {
		return f.runReceiver(path, ch)
	} else {
		return f.runSender(path, ch)
	}
}

func (f *FileTransfer) runSender(path string, ch TaskChannel) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	pos, err := file.Seek(0, os.SEEK_END)
	if err != nil {
		return err
	}
	ch.Send(pos)
	if _, err := file.Seek(0, os.SEEK_SET); err != nil {
		return err
	}

	defer file.Close()
	buf := make([]byte, transferBufferSize)
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		} else if n == 0 {
			continue
		}
		if err := ch.Send(buf[:n]); err != nil {
			return err
		}
	}
}

func (f *FileTransfer) runReceiver(path string, ch TaskChannel) (resErr error) {
	tempPath := path + fmt.Sprintf("%v", rand.Int63())
	outFile, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	sizeObj, err := ch.Receive()
	if err != nil {
		return fmt.Errorf("failed to read file size: %s", err)
	}
	size, ok := sizeObj.(int64)
	if !ok {
		return fmt.Errorf("invalid file size type: %T", sizeObj)
	}

	defer func() {
		if resErr != nil {
			outFile.Close()
			os.Remove(tempPath)
		}
	}()

	for {
		obj, err := ch.Receive()
		if err != nil {
			break
		}
		data, ok := obj.([]byte)
		if !ok {
			errObj, ok1 := obj.(error)
			if !ok1 {
				return fmt.Errorf("receiver got unexpected data type: %T", obj)
			}
			return fmt.Errorf("sender error: %s", errObj)
		}
		if _, err := outFile.Write(data); err != nil {
			return err
		}
	}

	if off, err := outFile.Seek(0, os.SEEK_CUR); err != nil {
		return err
	} else if off != size {
		return fmt.Errorf("expected file size %d but got size %d", size, off)
	}

	outFile.Close()
	return os.Rename(tempPath, path)
}
