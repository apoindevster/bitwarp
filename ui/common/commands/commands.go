package commands

import (
	"errors"
	"flag"
	"io"
	"strings"

	"github.com/apoindevster/bitwarp/commandclient"
	"github.com/apoindevster/bitwarp/proto"
)

// Callbacks encapsulates handlers invoked during command execution.
type Callbacks struct {
	Stdout   func([]byte)
	Stderr   func([]byte)
	Complete func(int32)
}

// RunExecutableCommand parses the exec argument string, streams stdout/stderr through callbacks,
// and returns the remote process exit code.
func RunExecutableCommand(argLine string, client *proto.CommandClient, cb Callbacks) (int32, error) {
	if client == nil {
		return -1, errors.New("command client is not initialized")
	}

	execPath, execArgs, err := parseExecArgs(argLine)
	if err != nil {
		return -1, err
	}

	dataChan := commandclient.MakeExecutableDataChan()
	completed := make(chan struct{})

	go func() {
		for {
			select {
			case out := <-dataChan.Stdout:
				if out != nil && cb.Stdout != nil {
					cb.Stdout(out)
				}
			case errBytes := <-dataChan.Stderr:
				if errBytes != nil && cb.Stderr != nil {
					cb.Stderr(errBytes)
				}
			case <-completed:
				return
			}
		}
	}()

	retCode := commandclient.RunExecutable(execPath, execArgs, &dataChan, client)
	completed <- struct{}{}

	if cb.Complete != nil {
		cb.Complete(retCode)
	}

	return retCode, nil
}

// ExecuteCommand interprets the top-level command and delegates to the appropriate handler.
// Currently, only the "exec" command is supported.
func ExecuteCommand(command string, args string, client *proto.CommandClient, cb Callbacks) (int32, error) {
	switch command {
	case "exec":
		return RunExecutableCommand(args, client, cb)
	case "":
		if cb.Complete != nil {
			cb.Complete(0)
		}
		return 0, nil
	default:
		if cb.Stderr != nil {
			cb.Stderr([]byte("Unrecognized command " + command + "\n"))
		}
		if cb.Complete != nil {
			cb.Complete(0)
		}
		return 0, nil
	}
}

func parseExecArgs(argLine string) (string, []string, error) {
	fields := strings.Fields(argLine)
	if len(fields) == 0 {
		return "", nil, errors.New("failed to get command to run")
	}

	fs := flag.NewFlagSet("ExecCommandSet", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(fields); err != nil {
		return "", nil, err
	}

	execPath := fs.Arg(0)
	execArgs := []string{}
	if fs.NArg() > 1 {
		execArgs = fs.Args()[1:]
	}

	return execPath, execArgs, nil
}
