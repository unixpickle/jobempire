package jobadmin

import (
	"errors"
	"sync"

	"github.com/unixpickle/jobempire/jobproto"
)

type jobRequest struct {
	Job *Job
	Res chan<- *LiveJob
	Err chan<- error
}

// A LiveMaster manages various aspects of an actively
// connected master connection.
type LiveMaster struct {
	master jobproto.Master

	shutdownLock sync.Mutex
	isShutdown   bool
	shutdown     chan struct{}

	newJobs chan jobRequest

	jobsLock sync.RWMutex
	jobs     []*LiveJob
	jobsNote nextNotifier
}

// RunLiveMaster creates a LiveMaster around an existing
// master connection.
func RunLiveMaster(m jobproto.Master) *LiveMaster {
	lm := &LiveMaster{
		master:   m,
		shutdown: make(chan struct{}),
		newJobs:  make(chan jobRequest),
	}
	go lm.runMaster()
	return lm
}

// SlaveInfo returns information about the slave.
func (l *LiveMaster) SlaveInfo() jobproto.SlaveInfo {
	return l.master.SlaveInfo()
}

// Accepting returns false if the master has been closed
// or is in the process of a graceful shutdown.
func (l *LiveMaster) Accepting() bool {
	select {
	case <-l.shutdown:
		return true
	default:
		return false
	}
}

// Running returns whether or not the master is fully
// disconnected.
func (l *LiveMaster) Running() bool {
	return l.jobsNote.Closed()
}

// Shutdown performs a graceful shutdown of the master,
// allowing it to complete all running tasks before fully
// disconnecting.
func (l *LiveMaster) Shutdown() {
	l.shutdownLock.Lock()
	defer l.shutdownLock.Unlock()
	if !l.isShutdown {
		close(l.shutdown)
		l.isShutdown = true
	}
}

// Cancel performs an abrupt shutdown of the master by
// closing its underlying connection.
// The underlying jobs may still take some time to fail.
func (l *LiveMaster) Cancel() {
	l.master.Close()
}

// RunJob queues up a job to be run on the master.
func (l *LiveMaster) RunJob(job *Job) (*LiveJob, error) {
	if !l.Accepting() {
		return nil, errors.New("master cannot accept new jobs")
	}
	resChan := make(chan *LiveJob, 1)
	errChan := make(chan error, 1)
	select {
	case l.newJobs <- jobRequest{job, resChan, errChan}:
		return <-resChan, <-errChan
	case <-l.shutdown:
		return nil, errors.New("master cannot accept new jobs")
	}
}

// JobCount returns the number of jobs which have been
// started on this master.
func (l *LiveMaster) JobCount() int {
	l.jobsLock.RLock()
	defer l.jobsLock.RUnlock()
	return len(l.jobs)
}

// Jobs returns a sub-range of jobs run on the master.
// It is like LiveJob.Tasks or LiveTask.LogEntries.
func (l *LiveMaster) Jobs(start, end int) []*LiveJob {
	l.jobsLock.RLock()
	defer l.jobsLock.RUnlock()
	return l.jobs[start:end]
}

// WaitJobs waits for a new job to be started, or for
// the master to be closed.
// It behaves like LiveTask.WaitLog.
func (l *LiveMaster) WaitJobs(n int, cancel <-chan struct{}) bool {
	return l.jobsNote.Wait(n, cancel)
}

// Wait waits until the master has been closed.
//
// The cancel argument specifies an optional channel which
// cancels the wait if it is closed.
func (l *LiveMaster) Wait(cancel <-chan struct{}) {
	l.jobsNote.WaitClose(cancel)
}

func (l *LiveMaster) runMaster() {
	go func() {
		l.master.Wait()
		l.Shutdown()
	}()

	defer func() {
		defer l.master.Close()
		defer l.jobsNote.Close()
		l.jobsLock.Lock()
		jobs := l.jobs
		l.jobsLock.Unlock()
		for _, j := range jobs {
			j.Wait(nil)
		}
	}()

	for {
		// If there was a shutdown, but also a lot of jobs
		// pending, we want to be sure to shutdown fast.
		select {
		case <-l.shutdown:
			return
		default:
		}

		select {
		case jobReq := <-l.newJobs:
			nextJob, err := RunLiveJob(l.master, jobReq.Job)
			jobReq.Res <- nextJob
			jobReq.Err <- err
			if err == nil {
				l.jobsLock.Lock()
				l.jobs = append(l.jobs, nextJob)
				l.jobsLock.Unlock()
				l.jobsNote.Notify()
			}
		case <-l.shutdown:
			return
		}
	}
}
