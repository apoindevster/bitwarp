package commandclient

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/apoindevster/bitwarp/proto"
)

func FileDownload(srcPath string, destPath string, client *proto.CommandClient) error {
	var f *os.File = nil
	var w *bufio.Writer = nil

	ctx := context.Background()
	stream, err := (*client).FileDownload(ctx, &proto.FileChunk{Path: srcPath})
	if err != nil {
		fmt.Printf("File Upload failed to start with error: %v\n", err)
		return err
	}

	for {
		m, err := stream.Recv()

		if err == io.EOF {
			return nil
		} else if err != nil {
			fmt.Printf("Failed file upload with err: %v\n", err)
			return err
		}

		// We have a message
		if f == nil {
			f, err = os.Create(destPath)
			if err != nil {
				fmt.Printf("Failed to create file with error %v", err)
				return err
			}
			w = bufio.NewWriter(f)
			defer w.Flush()
		}

		_, err = w.Write(m.GetChunk())
		if err != nil {
			fmt.Printf("Failed to write data to file upload: %v", err)
			return err
		}
	}
}

func FileUpload(srcPath string, destPath string, client *proto.CommandClient) error {
	expPath := os.ExpandEnv(srcPath)

	info, err := os.Stat(expPath)
	if err != nil {
		fmt.Printf("Failed to stat file with error: %v\n", err)
		return err
	}

	if info.IsDir() {
		fmt.Printf("Path provided is a directory... Please provide a file path\n")
		return os.ErrNotExist
	}

	ctx := context.Background()
	stream, err := (*client).FileUpload(ctx)
	if err != nil {
		fmt.Printf("File Upload failed to start with error: %v\n", err)
		return err
	}
	defer stream.CloseSend()

	var size int32 = 1024 * 1000
	rem := info.Size()

	f, err := os.Open(expPath)
	if err != nil {
		log.Fatalf("Failed to open file with error: %v", err)
		return err
	}

	for rem > 0 {
		if rem < int64(size) {
			size = int32(rem)
		}

		fbytes := make([]byte, size)

		_, err := io.ReadAtLeast(f, fbytes, int(size))

		if err != nil {
			fmt.Printf("Failed to read at least %d bytes: %v\n", size, err)
			return err
		}

		err = stream.Send(&proto.FileChunk{Path: destPath, Chunk: fbytes})
		if err != nil {
			fmt.Printf("Failed to send file chunk with error: %v\n", err)
			return err
		}

		rem = rem - int64(size)
	}
	stream.CloseAndRecv()
	return nil
}
