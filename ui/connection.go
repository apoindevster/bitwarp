package main

import (
	"errors"
	"strconv"

	"github.com/apoindevster/bitwarp/commandclient"
	"github.com/apoindevster/bitwarp/proto"
	newconn "github.com/apoindevster/bitwarp/ui/newconn"
	"google.golang.org/grpc"
)

func CreateNewConnection(params newconn.NewConnParams) (*grpc.ClientConn, *proto.CommandClient, error) {
	if params.Ip == "" || params.Port < 0 || params.Port > 65535 {
		return nil, nil, errors.New("invalid input params for new connection")
	}

	conn, client, err := commandclient.ConnectToServer(params.Ip + ":" + strconv.Itoa(params.Port))
	if err != nil {
		return nil, nil, errors.New("failed to connect to server")
	}

	return conn, &client, nil
}
