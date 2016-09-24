package jobproto

// A MasterConn provides control over the master side of
// a master-slave connection.
type MasterConn interface {
	// StartJob creates a new job over the connection.
	// Multiple jobs may be running simultaneously.
	StartJob() (MasterJob, error)

	// Terminate terminates the connection immediately.
	// If any jobs were running, the slave will be left
	// to handle the cleanup.
	// All created jobs and tasks will fail with an error
	// if they try to communicate with the slave.
	Terminate() error
}

// A MasterJob provides control over the master side of
// a job.
type MasterJob interface {
	// ID returns an identifier for the job that is unique
	// within the underlying connection.
	ID() int64

	// Finish notifies the slave that the job has completed.
	// This will fail if any tasks are still running.
	Finish() error

	// Run runs the task in the context of the given job.
	// It blocks until the task has completed running.
	// If the task finishes with an error, that error is
	// returned.
	// Multiple tasks may be run on a job simultaneously.
	Run(t Task) error
}
