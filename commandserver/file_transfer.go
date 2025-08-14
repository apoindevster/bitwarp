package commandserver

import (
	"bufio"
	"io"
	"os"

	"github.com/apoindevster/bitwarp/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Server) FileDownload(pathChunk *proto.FileChunk, stream grpc.ServerStreamingServer[proto.FileChunk]) error {
	expPath := os.ExpandEnv(pathChunk.GetPath())

	info, err := os.Stat(expPath)
	if err != nil {
		Logger.Warnf("Failed to stat file with error: %v\n", err)
		return err
	}

	if info.IsDir() {
		Logger.Warnf("Path provided is a directory... Please provide a file path\n")
		return os.ErrNotExist
	}

	var size int32 = 1024 * 1000
	rem := info.Size()

	f, err := os.Open(expPath)
	if err != nil {
		Logger.Warnf("Failed to open file with error: %v", err)
		return err
	}

	for rem > 0 {
		if rem < int64(size) {
			size = int32(rem)
		}

		fbytes := make([]byte, size)

		_, err := io.ReadAtLeast(f, fbytes, int(size))

		if err != nil {
			Logger.Warnf("Failed to read at least %d bytes: %v\n", size, err)
			return err
		}

		err = stream.Send(&proto.FileChunk{Path: expPath, Chunk: fbytes})
		if err != nil {
			Logger.Warnf("Failed to send file chunk with error: %v\n", err)
			return err
		}

		rem = rem - int64(size)
	}
	return nil
}

func (s *Server) FileUpload(stream grpc.ClientStreamingServer[proto.FileChunk, emptypb.Empty]) error {
	var f *os.File = nil
	var w *bufio.Writer = nil
	for {
		m, err := stream.Recv()

		if err == io.EOF {
			return err
		} else if err != nil {
			Logger.Warnf("Failed file upload with err: %v\n", err)
			return err
		}

		// We have a message
		if f == nil {
			f, err = os.Create(m.GetPath())
			if err != nil {
				Logger.Warnf("Failed to create file with error %v", err)
				return err
			}
			w = bufio.NewWriter(f)
			defer w.Flush()
		}

		_, err = w.Write(m.GetChunk())
		if err != nil {
			Logger.Warnf("Failed to write data to file upload: %v", err)
			return err
		}
	}
}
