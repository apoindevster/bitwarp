package shell

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/apoindevster/bitwarp/commandclient"
	"github.com/apoindevster/bitwarp/proto"
)

// The following Types are the possible custom tea.Msg types
// objects of these types get propagated back up to NotificationChan
type RunExecutableUpdate struct {
	appstring string
}

// For commands and flags to commands, use the flag package along with flagsets. This will allow for the subcommands that I am trying to accomplish
func RunExecutableCommand(command string, client *proto.CommandClient) error {
	cmdSet := flag.NewFlagSet("ExecCommandSet", -1)
	cmdSet.Parse(strings.Split(command, " "))

	if cmdSet.NArg() == 0 {
		return errors.New("failed to get command to run")
	}

	execpath := cmdSet.Arg(0)
	args := []string{}
	if cmdSet.NArg() > 1 {
		args = cmdSet.Args()[1:]
	}

	dataChan := commandclient.MakeExecutableDataChan()
	completed := make(chan struct{})
	go func() {
		for {
			select {
			case out := <-dataChan.Stdout:
				NotificationChan <- RunExecutableUpdate{appstring: string(out)}
			case err := <-dataChan.Stderr:
				NotificationChan <- RunExecutableUpdate{appstring: string(err)}
			case <-completed:
				return
			}
		}
	}()

	// The following line could be expanded to use the return item as the return code of the command and return that back to the user
	commandclient.RunExecutable(execpath, args, &dataChan, client)
	completed <- struct{}{}
	return nil
}

func ExecuteCommand(command string, args string, client *proto.CommandClient) error {
	switch command {
	case "exec":
		return RunExecutableCommand(args, client)
	case "upload":
		// TODO file upload: Awaiting progress bar in new window that will keep track of all of the commands that have been run by a client
		return nil
	case "download":
		// TODO file download: Awaiting progress bar in new window that will keep track of all of the commands that have been run by the client
		return nil
	default:
		// For now, just append the invalid to the history as a RunExecutableUpdate
		NotificationChan <- RunExecutableUpdate{appstring: fmt.Sprintf("Unrecognized command %s\n", command)}
		return nil
	}
}
