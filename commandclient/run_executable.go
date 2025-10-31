package commandclient

import (
	"context"
	"errors"
	"io"
	"log"

	proto "github.com/apoindevster/bitwarp/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func RunExecutable(ctx context.Context, command string, args []string, dataChan *ExecutableDataChan, client *proto.CommandClient) int32 {
	if ctx == nil {
		ctx = context.Background()
	}

	stream, err := (*client).RunExecutable(ctx)
	if err != nil {
		log.Fatalf("RunCommand failed with error: %v", err)
	}

	waitc := make(chan int32, 1)

	go func() {
		var returnCode int32
		for {
			r, err := stream.Recv()
			if err == io.EOF {
				// Finished Reading
				waitc <- returnCode
				return
			} else if err != nil {
				if status.Code(err) == codes.Canceled || status.Code(err) == codes.DeadlineExceeded || errors.Is(err, context.Canceled) {
					waitc <- -2
				} else {
					waitc <- -1
				}
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
	if err := stream.Send(&proto.RunExecutableInput{Options: options}); err != nil {
		return -1
	}

	for {
		select {
		case input := <-dataChan.Stdin:
			if err := stream.Send(&proto.RunExecutableInput{Stdin: input}); err != nil {
				return -1
			}
		case <-ctx.Done():
			stream.CloseSend()
			return -2
		case retCode := <-waitc:
			stream.CloseSend()
			return retCode
		}
	}
}
