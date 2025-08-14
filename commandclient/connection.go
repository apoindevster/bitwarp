package commandclient

import (
	"log"

	proto "github.com/apoindevster/bitwarp/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ConnectToServer(address string) (proto.CommandClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return nil, err
	}
	return proto.NewCommandClient(conn), nil
}
