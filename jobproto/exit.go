package jobproto

import (
	"encoding/gob"
	"os"
)

func init() {
	gob.Register(Exit{})
}

// Exit is a Task which exits the slave program.
type Exit struct{}

func (_ Exit) RunMaster(ch TaskChannel) error {
	return nil
}

func (_ Exit) RunSlave(root string, ch TaskChannel) error {
	os.Exit(1)
	return nil
}
