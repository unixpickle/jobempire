package jobadmin

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/unixpickle/jobempire/jobproto"
)

// A Task wraps one of the tasks from the jobproto package
// and provides JSON marshaling functionality.
type Task struct {
	Task jobproto.Task
}

// Copy creates a deep copy of the Task.
// It fails if the Task is not one of the supported types
// supported by MarshalJSON.
func (t *Task) Copy() (*Task, error) {
	marshaled, err := t.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var res Task
	if err := res.UnmarshalJSON(marshaled); err != nil {
		return nil, err
	}
	return &res, nil
}

// MarshalJSON marshal's the internal task, given that it
// is one of the following types:
//
//     - *jobproto.FileTransfer
//     - *jobproto.GoRun
//     - *jobproto.Exit
//
// The resulting JSON object has fields for each of those
// types (e.g. a field named "GoRun").
// Exactly one of said fields will be non-null and contain
// the JSON-marshaled version of the task.
//
// This will fail if t.Task is not a supported type.
func (t *Task) MarshalJSON() ([]byte, error) {
	var res marshalTask
	switch task := t.Task.(type) {
	case *jobproto.FileTransfer:
		res.FileTransfer = task
	case *jobproto.GoRun:
		res.GoRun = task
	case *jobproto.Exit:
		res.Exit = task
	default:
		return nil, fmt.Errorf("unsupported task type: %T", t.Task)
	}
	return json.Marshal(res)
}

// UnmarshalJSON performs the inverse of MarshalJSON.
func (t *Task) UnmarshalJSON(d []byte) error {
	var mt marshalTask
	if err := json.Unmarshal(d, &mt); err != nil {
		return err
	}
	switch true {
	case mt.FileTransfer != nil:
		t.Task = mt.FileTransfer
	case mt.GoRun != nil:
		t.Task = mt.GoRun
	case mt.Exit != nil:
		t.Task = mt.Exit
	default:
		return errors.New("missing task to unmarshal")
	}
	return nil
}

type marshalTask struct {
	FileTransfer *jobproto.FileTransfer
	GoRun        *jobproto.GoRun
	Exit         *jobproto.Exit
}
