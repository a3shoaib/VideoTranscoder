# Generating proto
protoc -I=. --go_out=. --go-grpc_out=. ./largefile.proto

# Running the server
Ensure the docker Daemon is running
cd server
docker build --tag=transcoder:latest . && docker run -it -p 9000:9000 transcoder:latest

# Running the client
go install projects/arshoaib/largefile-streaming/client && ~/go/bin/client

## Instructions

Follow this guide: https://developers.google.com/protocol-buffers/docs/gotutorial
- Install protoc using homebrew
- Install Golang
- Install Golang protoc plugins:
`$ go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28`
`$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2`
- Ensure that you are in the base directory, then run the following command:
`protoc -I=. --go_out=. --go-grpc_out=. ./videomux.proto`


