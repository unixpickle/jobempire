package jobadmin

import "fmt"

// A Job stores static information about a job and about
// the way in which the job should be deployed.
type Job struct {
	// ID stores the unique ID of the Job.
	ID string

	// Name stores a human-readable name for the job.
	// This needn't be unique in a pool of jobs, but
	// uniqueness would probably help users.
	Name string

	// Tasks stores a sequential list of task descriptions
	// to be run in the job.
	Tasks []*Task

	// MaxInstances limits the number of job instances the
	// scheduler may create at once.
	//
	// A value of 0 means that the scheduler is free to
	// launch the job as much as it wants.
	//
	// This value limits the scheduler, but not the admin.
	// The administrator can manually create more than
	// MaxInstances instances of a job at once.
	MaxInstances int

	// Priority specifies how often this job should be run
	// by the automated scheduler.
	// A higher priority value means that the job is run
	// with a higher probability.
	// A priority of 0 means that the task will not be
	// scheduled.
	Priority int

	// NumCPU specifies the maximum number of CPUs this job
	// will demand.
	//
	// The scheduler will never add a job to a slave if the
	// slave is already running something and adding the
	// job will push the slave's total MaxCPU sum to a value
	// greater than the slave's MaxProcs value.
	//
	// This may be 0 for jobs that are not CPU-bound.
	NumCPU int
}

// Copy creates a deep copy of the Job.
// It will fail if any of the underlying Tasks cannot be
// copied with their Copy methods.
func (j *Job) Copy() (*Job, error) {
	res := *j
	res.Tasks = make([]*Task, len(j.Tasks))
	for i, t := range j.Tasks {
		var e error
		res.Tasks[i], e = t.Copy()
		if e != nil {
			return nil, fmt.Errorf("task %d: %s", i, e)
		}
	}
	return &res, nil
}

// Unbounded returns true if the job will be scheduled
// an infinite number of times and cause problems.
func (j *Job) Unbounded() bool {
	return j.NumCPU == 0 && j.Priority > 0 && j.MaxInstances == 0
}
