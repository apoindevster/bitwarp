package main

import (
	"fmt"

	"github.com/apoindevster/bitwarp/commandclient"
	"github.com/apoindevster/bitwarp/proto"
	"github.com/apoindevster/bitwarp/shell"
	tea "github.com/charmbracelet/bubbletea"
)

var client proto.CommandClient
var Prog *tea.Program

// I think I will want to pass a

func main() {
	var err error
	client, err = commandclient.ConnectToServer("localhost:8090")
	if err != nil {
		return
	}

	notif := make(chan tea.Msg)

	Prog = tea.NewProgram(shell.New(&client, notif), tea.WithAltScreen())
	if _, err := Prog.Run(); err != nil {
		fmt.Printf("Failed to run tui interface with error: %v\n", err)
		return
	}
}
