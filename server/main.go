package main

import (
	"fmt"
	"log"
	"net"

	pb "projects/arshoaib/largefile-streaming/server/exports/compiled_proto"

	"google.golang.org/grpc"
)

func main() {
	port := 9000
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("Listening on port %d...\n", port)

	s := Server{}

	grpcServer := grpc.NewServer()

	pb.RegisterVideoTranscoderServiceServer(grpcServer, &s)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
