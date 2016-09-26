package jobproto

const transferBufferSize = 65536

// A Task implements a unit of work.
// When a Task is sent from a master to a slave, it will
// serialized using the gob package.
type Task interface {
	// RunMaster runs the master side of the task.
	RunMaster(ch TaskChannel) error

	// RunSlave runs the slave side of the task.
	// The rootDir specifies the root directory for files
	// related to this task.
	RunSlave(rootDir string, ch TaskChannel) error
}

// A TaskChannel facilitates communication between a
// master task and a slave task.
type TaskChannel interface {
	// Send sends a message to the other side of the task.
	// It blocks until the message has been sent.
	// It fails with an error if the other side is already
	// done.
	Send(msg interface{}) error

	// Receive receives the next message from the other side
	// of the task.
	// It blocks until a message is received.
	// It returns an error after all messages from the other
	// side have been received and the other side is done.
	Receive() (interface{}, error)
}
