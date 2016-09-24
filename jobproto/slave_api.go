package jobproto

// A SlaveConn provides a stream of jobs from a master.
type SlaveConn interface {
	// NextJob receives the next job from the master.
	// This will return an error if the remote end has
	// terminated the connection.
	NextJob() (SlaveJob, error)
}

// A SlaveJob provides a stream of tasks from a master.
type SlaveJob interface {
	// ID returns an identifier for the job that is
	// unique for the underlying SlaveConn.
	ID() int64

	// RunNext runs the next task from the master.
	// It returns an error if the task fails to run.
	// It returns io.EOF if the job is complete.
	RunNext(rootDir string) error
}
