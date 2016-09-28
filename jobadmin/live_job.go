package jobadmin

import (
	"fmt"
	"sync"

	"github.com/unixpickle/jobempire/jobproto"
)

// A LiveJob manages a running or previously-run instance
// of a job, including its live tasks.
type LiveJob struct {
	job       *Job
	masterJob jobproto.MasterJob

	tasksLock sync.RWMutex
	tasks     []*LiveTask
	tasksNote nextNotifier

	resLock  sync.RWMutex
	resError error
}

// RunLiveJob launches a job on the Master.
// The job's tasks will automatically be run as LiveTasks.
func RunLiveJob(m jobproto.Master, j *Job) (*LiveJob, error) {
	jobCopy, err := j.Copy()
	if err != nil {
		return nil, fmt.Errorf("copy Job: %s", err)
	}

	masterJob, err := m.StartJob()
	if err != nil {
		return nil, fmt.Errorf("start job: %s", err)
	}

	lj := &LiveJob{
		job:       jobCopy,
		masterJob: masterJob,
	}
	go lj.runJob()
	return lj, nil
}

// Job returns the job.
// The caller should not modify the result.
// This will never change, even if the original job is
// modified by the administrator.
func (l *LiveJob) Job() *Job {
	return l.job
}

// Running returns whether or not the job is running.
func (l *LiveJob) Running() bool {
	return !l.tasksNote.Closed()
}

// Cancel requests closes the underlying job, triggering
// the job to end.
// This may not have an immediate effect, since the job
// will only end once the current task discovers that the
// job has been disconnected.
func (l *LiveJob) Cancel() {
	l.masterJob.Close()
}

// TaskCount returns the number of tasks which have been
// completed or started.
// This does not count tasks which have not (or will not)
// be run.
func (l *LiveJob) TaskCount() int {
	l.tasksLock.RLock()
	defer l.tasksLock.RUnlock()
	return len(l.tasks)
}

// Tasks returns a sub-range of the LiveTasks which have
// been run or are running.
// Range indices work like slice indices, where 0 is the
// the first task.
//
// The range must be within bounds, which is possible to
// ensure by using the value from TaskCount as a limit.
//
// The caller should not modify the result.
func (l *LiveJob) Tasks(start, end int) []*LiveTask {
	l.tasksLock.RLock()
	defer l.tasksLock.RUnlock()
	return l.tasks[start:end]
}

// WaitTasks waits for a new task to be started, or for
// the job to complete.
// It behaves like LiveTask.WaitLog.
func (l *LiveJob) WaitTasks(n int, cancel <-chan struct{}) bool {
	return l.tasksNote.Wait(n, cancel)
}

// Wait waits until the job has finished running.
//
// The cancel argument specifies an optional channel which
// cancels the wait if it is closed.
func (l *LiveJob) Wait(cancel <-chan struct{}) {
	l.tasksNote.WaitClose(cancel)
}

// Error returns the first error that caused the job to
// fail, if there was one.
func (l *LiveJob) Error() error {
	l.resLock.RLock()
	defer l.resLock.RUnlock()
	return l.resError
}

func (l *LiveJob) runJob() {
	for _, t := range l.job.Tasks {
		lt, err := RunLiveTask(l.masterJob, t)
		if err != nil {
			l.done(err)
			return
		}
		l.tasksLock.Lock()
		l.tasks = append(l.tasks, lt)
		l.tasksLock.Unlock()
		l.tasksNote.Notify()
		lt.Wait(nil)
		if err := lt.Error(); err != nil {
			l.done(fmt.Errorf("task error: %s", err))
			return
		}
	}
	l.done(nil)
}

func (l *LiveJob) done(e error) {
	if e == nil {
		e = l.masterJob.Close()
	} else {
		l.masterJob.Close()
	}
	if e != nil {
		l.resLock.Lock()
		l.resError = e
		l.resLock.Unlock()
	}
	l.tasksNote.Close()
}
