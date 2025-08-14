package commandclient

import (
	"context"
	"io"
	"log"

	proto "github.com/apoindevster/bitwarp/proto"
)

type ExecutableDataChan struct {
	Stdout chan []byte
	Stderr chan []byte
	Stdin  chan []byte
}

func MakeExecutableDataChan() ExecutableDataChan {
	return ExecutableDataChan{
		Stdout: make(chan []byte),
		Stderr: make(chan []byte),
		Stdin:  make(chan []byte),
	}
}

func RunExecutable(command string, args []string, dataChan *ExecutableDataChan, client *proto.CommandClient) int32 {
	ctx := context.Background()
	stream, err := (*client).RunExecutable(ctx)
	if err != nil {
		log.Fatalf("RunCommand failed with error: %v", err)
	}

	waitc := make(chan int32)

	go func() {
		var returnCode int32
		for {
			r, err := stream.Recv()
			if err == io.EOF {
				// Finished Reading
				waitc <- returnCode
				return
			} else if err != nil {
				waitc <- -1
				return
			}

			returnCode = r.GetReturnCode()
			stdout := r.GetStdout()
			stderr := r.GetStderr()
			if stdout != nil {
				dataChan.Stdout <- stdout
			}
			if stderr != nil {
				dataChan.Stderr <- stderr
			}
		}
	}()

	options := &proto.RunExecutableOptions{Command: command, Args: args}
	stream.Send(&proto.RunExecutableInput{Options: options})

	for {
		select {
		case input := <-dataChan.Stdin:
			stream.Send(&proto.RunExecutableInput{Stdin: input})
		case retCode := <-waitc:
			stream.CloseSend()
			return retCode
		}
	}
}
