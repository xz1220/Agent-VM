// Package runlog persists run history (RunRecord) per PRD §4.4.
package runlog

import (
	"github.com/xz1220/agent-vm/internal/app/model"
)

// Log is the contract for run history persistence.
type Log interface {
	Append(record model.RunRecord) error
	List(limit int) ([]model.RunRecord, error)
}

// FSLog is the on-disk default implementation.
type FSLog struct{ Dir string }

func New(dir string) *FSLog { return &FSLog{Dir: dir} }

func (l *FSLog) Append(model.RunRecord) error              { return nil }
func (l *FSLog) List(limit int) ([]model.RunRecord, error) { return nil, nil }
