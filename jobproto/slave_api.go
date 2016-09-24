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
	// RunTasks runs the tasks from the master.
	RunTasks(rootDir string)
}
