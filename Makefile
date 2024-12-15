gencode:
	protoc --go_out=. --go-grpc_out=proto proto/hashlists.proto

test:
	go test ./...
