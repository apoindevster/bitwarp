package commandserver

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"

	"github.com/apoindevster/bitwarp/proto"
	"google.golang.org/grpc"
)

func readOutputPipe(pipe io.ReadCloser, output chan []byte) {
	buf := make([]byte, 1024)
	for {
		read, err := pipe.Read(buf)
		if err != nil {
			break
		}
		output <- buf[:read]
	}
	close(output)
}

func (s *Server) RunExecutable(stream grpc.BidiStreamingServer[proto.RunExecutableInput, proto.RunExecutableResult]) error {
	options, err := stream.Recv()
	if err != nil {
		Logger.Warn("Failed to get which command to run")
		return err
	}

	Logger.Infof("Starting command %s %s", options.GetOptions().Command, strings.Join(options.GetOptions().Args, " "))
	command := exec.Command(options.GetOptions().Command, options.GetOptions().Args...)
	ctx := stream.Context()

	stdout, err := command.StdoutPipe()
	if err != nil {
		stream.Send(&proto.RunExecutableResult{Stderr: []byte(fmt.Sprintf("Failed to create stdout pipe with error: %v", err)), ReturnCode: -1})
		Logger.Warn("Failed to create pipe for stdout")
		return nil
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		stream.Send(&proto.RunExecutableResult{Stderr: []byte(fmt.Sprintf("Failed to create stderr pipe with error: %v", err)), ReturnCode: -1})
		Logger.Warn("Failed to create pipe for stderr")
		return nil
	}

	var returnCode int = 0
	err = command.Start()
	if err != nil {
		stream.Send(&proto.RunExecutableResult{Stderr: []byte(fmt.Sprintf("Failed to start command with error: %v", err)), ReturnCode: -1})
		Logger.Warnf("Failed to start the command: %s with args: %s", options.GetOptions().Command, options.GetOptions().Args)
		return nil
	}

	sout := make(chan []byte)
	serr := make(chan []byte)
	go readOutputPipe(stdout, sout)
	go readOutputPipe(stderr, serr)

	stdoutFinished := false
	stderrFinished := false
	cancelled := false
finished:
	for {
		select {
		case output, ok := <-sout:
			if ok {
				stream.Send(&proto.RunExecutableResult{Stdout: output})
			} else {
				stdoutFinished = true
			}
		case output, ok := <-serr:
			if ok {
				stream.Send(&proto.RunExecutableResult{Stderr: output})
			} else {
				stderrFinished = true
			}
		case <-ctx.Done():
			cancelled = true
			if command.Process != nil && command.ProcessState == nil {
				_ = command.Process.Kill()
			}
		}

		if (stdoutFinished && stderrFinished) || command.ProcessState != nil {
			break finished
		}
	}
	Logger.Infof("Finishing command %s %s", options.GetOptions().Command, strings.Join(options.GetOptions().Args, " "))
	err = command.Wait()
	if err != nil {
		if cancelled {
			returnCode = -2
			stream.Send(&proto.RunExecutableResult{ReturnCode: int32(returnCode)})
			return nil
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			// We were able to cast the error. Now see if we can get the wait status
			if ws, ok := exitError.Sys().(syscall.WaitStatus); ok {
				// We were able to get the Wait status so we should be able to get the return code
				returnCode = ws.ExitStatus()
			} else {
				stream.Send(&proto.RunExecutableResult{Stderr: []byte(fmt.Sprintf("Failed to get exit status from the command with error: %v", err)), ReturnCode: -1})
				Logger.Warn("Failed to get the Exit status from the command.")
				return nil
			}
		} else {
			stream.Send(&proto.RunExecutableResult{Stderr: []byte(fmt.Sprintf("Failed to cast the error to an ExitError with error: %v", err)), ReturnCode: -1})
			return nil
		}
	}

	stream.Send(&proto.RunExecutableResult{ReturnCode: int32(returnCode)})
	return nil
}
