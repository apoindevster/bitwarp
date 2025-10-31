package jobs

import (
	"context"

	"github.com/google/uuid"
)

type Stream int

const (
	StreamStdout Stream = iota
	StreamStderr
)

type Source string

const (
	SourceShell  Source = "shell"
	SourceRunAll Source = "runall"
)

type StartedMsg struct {
	JobID        uuid.UUID
	ConnectionID uuid.UUID
	Command      string
	Source       Source
	Cancel       context.CancelFunc
}

type OutputMsg struct {
	JobID        uuid.UUID
	ConnectionID uuid.UUID
	Data         string
	Stream       Stream
}

type CompletedMsg struct {
	JobID        uuid.UUID
	ConnectionID uuid.UUID
	ReturnCode   int32
}
