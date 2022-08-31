# VideoMux

## Instructions

Follow this guide: https://developers.google.com/protocol-buffers/docs/gotutorial
- Install protoc using homebrew
- Install Golang
- Install Golang protoc plugins:
`$ go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28`
`$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2`
- Ensure that you are in the base directory, then run the following command:
`protoc -I=. --go_out=. --go-grpc_out=. ./videomux.proto`



## Planned commands
videomux create_account
videomux login
videomux logout
videomux send ....
videomux receive ....
videomux list --options (sort)
videomux convert formatA to formatB
videomux rename videoA to videoB
videomux search videoA

