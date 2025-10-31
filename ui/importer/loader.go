package importer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type CommandBatch struct {
	Version  string              `json:"version"`
	Metadata map[string]any      `json:"metadata,omitempty"`
	Commands []CommandDefinition `json:"commands"`
}

type CommandDefinition struct {
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Async          bool              `json:"async,omitempty"`
	Exec           ExecSpec          `json:"exec"`
	Expect         Expectation       `json:"expect"`
	TimeoutSeconds *int              `json:"timeoutSeconds,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Stdin          string            `json:"stdin,omitempty"`
}

type ExecSpec struct {
	Command    string   `json:"command"`
	Args       []string `json:"args,omitempty"`
	WorkingDir string   `json:"workingDir,omitempty"`
}

type Expectation struct {
	Mode  string `json:"mode"`
	Code  *int   `json:"code,omitempty"`
	Codes []int  `json:"codes,omitempty"`
	Min   *int   `json:"min,omitempty"`
	Max   *int   `json:"max,omitempty"`
}

func LoadCommandBatch(path string) (*CommandBatch, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open batch file: %w", err)
	}
	defer f.Close()

	var batch CommandBatch
	if err := json.NewDecoder(f).Decode(&batch); err != nil {
		return nil, fmt.Errorf("failed to decode batch file: %w", err)
	}

	if batch.Version == "" {
		return nil, errors.New("batch missing version")
	}
	if len(batch.Commands) == 0 {
		return nil, errors.New("batch must contain at least one command")
	}

	for i := range batch.Commands {
		if err := batch.Commands[i].normalize(); err != nil {
			return nil, fmt.Errorf("command %d invalid: %w", i, err)
		}
	}

	return &batch, nil
}

func (cmd *CommandDefinition) normalize() error {
	cmd.Exec.Command = strings.TrimSpace(cmd.Exec.Command)
	if cmd.Exec.Command == "" {
		return errors.New("exec.command is required")
	}
	if cmd.Name == "" {
		cmd.Name = cmd.Exec.Command
	}
	if cmd.TimeoutSeconds != nil && *cmd.TimeoutSeconds <= 0 {
		return errors.New("timeoutSeconds must be greater than zero")
	}
	if cmd.Expect.Mode == "" {
		cmd.Expect.Mode = "eq"
		if cmd.Expect.Code == nil {
			def := 0
			cmd.Expect.Code = &def
		}
	}
	cmd.Expect.Mode = strings.ToLower(cmd.Expect.Mode)
	switch cmd.Expect.Mode {
	case "any":
	case "eq", "ne":
		if cmd.Expect.Code == nil {
			return fmt.Errorf("expect.mode %s requires code", cmd.Expect.Mode)
		}
	case "in", "notin":
		if len(cmd.Expect.Codes) == 0 {
			return fmt.Errorf("expect.mode %s requires non-empty codes", cmd.Expect.Mode)
		}
	case "range":
		if cmd.Expect.Min == nil || cmd.Expect.Max == nil {
			return errors.New("expect.range requires min and max")
		}
		if *cmd.Expect.Min > *cmd.Expect.Max {
			return errors.New("expect.range min cannot be greater than max")
		}
	default:
		return fmt.Errorf("unsupported expect.mode %s", cmd.Expect.Mode)
	}
	return nil
}

func (e Expectation) Evaluate(prev int32) bool {
	switch e.Mode {
	case "any", "":
		return true
	case "eq":
		if e.Code == nil {
			return prev == 0
		}
		return prev == int32(*e.Code)
	case "ne":
		if e.Code == nil {
			return prev != 0
		}
		return prev != int32(*e.Code)
	case "in":
		for _, c := range e.Codes {
			if prev == int32(c) {
				return true
			}
		}
		return false
	case "notin":
		for _, c := range e.Codes {
			if prev == int32(c) {
				return false
			}
		}
		return true
	case "range":
		if e.Min != nil && prev < int32(*e.Min) {
			return false
		}
		if e.Max != nil && prev > int32(*e.Max) {
			return false
		}
		return true
	default:
		return false
	}
}
