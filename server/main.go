package main

import (
	"net"

	"github.com/apoindevster/bitwarp/commandserver"
	"github.com/apoindevster/bitwarp/proto"
	grpc "google.golang.org/grpc"
)

func main() {
	if commandserver.SetupLogger("", false) != nil {
		commandserver.Logger.Fatal("failed to setup the proper logger")
		return
	}

	lis, err := net.Listen("tcp", ":8090")
	if err != nil {
		commandserver.Logger.Fatalf("failed to listen: %v", err)
		return
	}
	defer lis.Close()
	s := grpc.NewServer()
	proto.RegisterCommandServer(s, &commandserver.Server{})
	commandserver.Logger.Infof("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		commandserver.Logger.Fatalf("failed to serve: %v", err)
		return
	}
}
