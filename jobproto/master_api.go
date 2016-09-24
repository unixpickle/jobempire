package jobproto

// A MasterConn provides control over the master side of
// a master-slave connection.
type MasterConn interface {
	// StartJob creates a new job over the connection.
	// Multiple jobs may be running simultaneously.
	StartJob() (MasterJob, error)

	// Terminate terminates the connection immediately.
	// If any jobs were running, the slave and master
	// will be left to handle the cleanup.
	// All created jobs and tasks will fail with an error
	// when they try to communicate with the remote end.
	Terminate()
}

// A MasterJob provides control over the master side of
// a job.
type MasterJob interface {
	// Finish terminates the job.
	// If no tasks were running, it performs a graceful
	// shutdown of the job.
	// If tasks were running, their connections are closed
	// and they must handle the failure.
	Finish() error

	// Run runs a task in the context of the job.
	// It blocks until the task has completed on both the
	// master and the slave.
	// It returns an error if the task fails on either end,
	// or if the job is finished early.
	// Multiple tasks may be run on a job simultaneously.
	Run(t Task) error
}
