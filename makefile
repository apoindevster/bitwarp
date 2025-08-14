all:
	protoc --go_out=. --go-grpc_out=. commands.proto