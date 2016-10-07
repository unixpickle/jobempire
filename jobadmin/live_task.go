package jobadmin

import (
	"fmt"
	"sync"
	"time"

	"github.com/unixpickle/jobempire/jobproto"
)

// A LiveTask contains information about a running or
// previously run instance of a task, including its logged
// output and final status.
type LiveTask struct {
	task      *Task
	startTime time.Time

	logLock sync.RWMutex
	log     []jobproto.LogEntry
	logNote nextNotifier

	resLock  sync.RWMutex
	resError error
	endTime  time.Time
}

// RunLiveTask runs a Task and creates a LiveTask for it.
func RunLiveTask(j jobproto.MasterJob, t *Task) (*LiveTask, error) {
	taskCopy, err := t.Copy()
	if err != nil {
		return nil, fmt.Errorf("copy task: %s", err)
	}
	lt := &LiveTask{
		task:      taskCopy,
		startTime: time.Now(),
	}
	go lt.runTask(j)
	return lt, nil
}

// Task returns the task.
// The caller should not modify the result.
// This will never change, even if the original task is
// modified by the administrator.
func (l *LiveTask) Task() *Task {
	return l.task
}

// Running returns whether or not the task is running.
func (l *LiveTask) Running() bool {
	return !l.logNote.Closed()
}

// LogSize returns the current number of log entries.
func (l *LiveTask) LogSize() int {
	l.logLock.RLock()
	defer l.logLock.RUnlock()
	return len(l.log)
}

// LogEntries returns the log entries in the given range,
// where range indices work like slice indices, and where
// 0 is the first log entry.
//
// The range must be within bounds, which is possible to
// ensure by using the value from LogSize as a limit.
//
// The caller should not modify the result.
func (l *LiveTask) LogEntries(start, end int) []jobproto.LogEntry {
	l.logLock.RLock()
	defer l.logLock.RUnlock()
	return l.log[start:end]
}

// WaitLog waits for new log entries to arrive.
//
// The value of n should be the last value of LogSize seen
// by the caller.
// When there are more than n log entries (or when more log
// entries arrive), WaitLog returns true.
//
// The cancel argument specifies an optional channel which
// cancels the wait if it is closed.
//
// When the task finishes or the wait is cancelled before a
// new log entry arrives, this returns false.
//
// This should never be called with an n value less than the
// current log size.
func (l *LiveTask) WaitLog(n int, cancel <-chan struct{}) bool {
	return l.logNote.Wait(n, cancel)
}

// Wait waits until the tasks has finished running.
//
// The cancel argument specifies an optional channel which
// cancels the wait if it is closed.
func (l *LiveTask) Wait(cancel <-chan struct{}) {
	l.logNote.WaitClose(cancel)
}

// Error returns an error, if there was one, from when the
// task finished running.
func (l *LiveTask) Error() error {
	l.resLock.RLock()
	defer l.resLock.RUnlock()
	return l.resError
}

// StartTime returns the time when the task was started.
func (l *LiveTask) StartTime() time.Time {
	return l.startTime
}

// EndTime returns the time when the task finished.
// This is the zero value of time.Time if the task is not
// finished yet.
func (l *LiveTask) EndTime() time.Time {
	l.resLock.RLock()
	defer l.resLock.RUnlock()
	return l.endTime
}

func (l *LiveTask) runTask(j jobproto.MasterJob) {
	logChan := make(chan jobproto.LogEntry)
	go func() {
		for entry := range logChan {
			l.logLock.Lock()
			l.log = append(l.log, entry)
			l.logLock.Unlock()
			l.logNote.Notify()
		}
		l.logNote.Close()
	}()
	err := j.Run(l.task.Task, logChan)
	l.resLock.Lock()
	l.resError = err
	l.endTime = time.Now()
	l.resLock.Unlock()
	close(logChan)
}
