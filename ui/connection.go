package main

import (
	"errors"
	"strconv"

	"github.com/apoindevster/bitwarp/commandclient"
	newconn "github.com/apoindevster/bitwarp/ui/newconn"
	"google.golang.org/grpc"
)

func CreateNewConnection(params newconn.NewConnParams) (*grpc.ClientConn, error) {
	if params.Ip == "" || params.Port < 0 || params.Port > 65535 {
		return nil, errors.New("invalid input params for new connection")
	}

	conn, err := commandclient.ConnectToServer(params.Ip + ":" + strconv.Itoa(params.Port))
	if err != nil {
		return nil, errors.New("failed to connect to server")
	}

	return conn, nil
}
