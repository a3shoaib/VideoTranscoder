package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	pb "projects/arshoaib/largefile-streaming/client/exports/compiled_proto"

	"google.golang.org/grpc"
)

type CommandLineArgs struct {
	InputFilePath      string
	BucketName         string
	VideoCodecName     string
	AudioCodecName     string
	AwsRegion          string
	AwsAccessKeyId     string
	AwsSecretAccessKey string
}

func ReadInput() CommandLineArgs {
	commandLineArgs := CommandLineArgs{}

	fmt.Println("Enter the input file path.")
	fmt.Scanln(&commandLineArgs.InputFilePath)
	if len(commandLineArgs.InputFilePath) <= 0 {
		log.Fatalf("Must specify an input file path.")
	}

	fmt.Println("Enter the S3 Bucket name.")
	fmt.Scanln(&commandLineArgs.BucketName)
	if len(commandLineArgs.BucketName) <= 0 {
		log.Fatalf("Must specify an S3 bucket name.")
	}

	fmt.Println("Enter the output video codec name (h264, hevc, vp9, av1) or leave blank.")
	fmt.Scanln(&commandLineArgs.VideoCodecName)

	fmt.Println("Enter the output audio codec name (aac, flac, ac3) or leave blank.")
	fmt.Scanln(&commandLineArgs.AudioCodecName)

	fmt.Println("Enter the AWS Region (or leave blank to read from AWS_REGION environment variable).")
	fmt.Scanln(&commandLineArgs.AwsRegion)
	if len(commandLineArgs.AwsRegion) <= 0 {
		if commandLineArgs.AwsRegion = os.Getenv("AWS_REGION"); len(commandLineArgs.AwsRegion) <= 0 {
			log.Fatalf("Must specify an AWS region.")
		}
	}

	fmt.Println("Enter the AWS access key id (or leave blank to read from AWS_ACCESS_KEY_ID environment variable).")
	fmt.Scanln(&commandLineArgs.AwsAccessKeyId)
	if len(commandLineArgs.AwsAccessKeyId) <= 0 {
		if commandLineArgs.AwsAccessKeyId = os.Getenv("AWS_ACCESS_KEY_ID"); len(commandLineArgs.AwsAccessKeyId) <= 0 {
			log.Fatalf("Must specify an AWS region.")
		}
	}

	fmt.Println("Enter the AWS secret access key (or leave blank to read from AWS_SECRET_ACCESS_KEY environment variable).")
	fmt.Scanln(&commandLineArgs.AwsSecretAccessKey)
	if len(commandLineArgs.AwsSecretAccessKey) <= 0 {
		if commandLineArgs.AwsSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY"); len(commandLineArgs.AwsSecretAccessKey) <= 0 {
			log.Fatalf("Must specify an AWS region.")
		}
	}

	return commandLineArgs
}

func main() {
	// Establish the gRPC connection to the server on target <target>
	port := 9000
	target := fmt.Sprintf("localhost:%d", port)
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	fmt.Printf("Dialing port %d...\n", port)
	if err != nil {
		log.Fatalf("Did not connect to gRPC server: %s", err)
	}
	defer conn.Close()

	// Create the gRPC client.
	client := pb.NewVideoTranscoderServiceClient(conn)

	// Read input from command line.
	commandLineArgs := ReadInput()

	// Open the input video file for reading.
	inputFile, err := os.Open(commandLineArgs.InputFilePath)
	if err != nil {
		log.Fatalf("Could not open input file: %v", err)
	}
	defer inputFile.Close()

	// Send the RPC for SendVideoChunk and obtain the gRPC stream.
	stream, err := client.SendVideoChunk(context.Background())
	if err != nil {
		log.Fatalf("Error creating SendVideoChunk stream: %v", err)
	}

	// Send the header.
	header := &pb.VideoChunk{
		Chunk: &pb.VideoChunk_Header{
			Header: &pb.TranscoderHeaderInformation{
				FileName: filepath.Base(commandLineArgs.InputFilePath),
				AwsCredentials: &pb.AWSCredentials{
					BucketName:      commandLineArgs.BucketName,
					Region:          commandLineArgs.AwsRegion,
					AccessKeyId:     commandLineArgs.AwsAccessKeyId,
					SecretAccessKey: commandLineArgs.AwsSecretAccessKey,
				},
				OutputVideoCodec: GetEnumFromVideoCodecName(commandLineArgs.VideoCodecName),
				OutputAudioCodec: GetEnumFromAudioCodecName(commandLineArgs.AudioCodecName),
			},
		},
	}
	err = stream.Send(header)
	if err != nil {
		log.Fatalf("Error sending the header: %v", err)
	}

	// Read the file in chunks, and send each chunk to the stream.
	reader := bufio.NewReader(inputFile)
	chunkSize := 65536 // 2^16
	buf := make([]byte, chunkSize)

	fmt.Println("Sending video file...")
	for {
		_, err := reader.Read(buf)

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("Could not read chunk: %v ", err)
		}

		videoChunk := pb.VideoChunk{
			Chunk: &pb.VideoChunk_ChunkData{
				ChunkData: buf,
			},
		}

		if err = stream.Send(&videoChunk); err != nil {
			log.Fatalf("Could not send chunk: %v", err)
		}
	}

	// Get the response.
	fmt.Println("Sent video file.")
	_, err = stream.CloseAndRecv()

	if err != nil {
		log.Fatalf("Transcoding error: %v", err)
	}
	fmt.Println("Done transcoding and uploaded to S3!")
}

func GetEnumFromVideoCodecName(codec string) *pb.VideoCodec {
	if codec == "h264" {
		return pb.VideoCodec_H264.Enum()
	} else if codec == "av1" {
		return pb.VideoCodec_AV1.Enum()
	} else if codec == "hevc" {
		return pb.VideoCodec_HEVC.Enum()
	} else if codec == "vp9" {
		return pb.VideoCodec_VP9.Enum()
	}
	return pb.VideoCodec_COPY_VCODEC.Enum()
}

func GetEnumFromAudioCodecName(codec string) *pb.AudioCodec {
	if codec == "aac" {
		return pb.AudioCodec_AAC.Enum()
	} else if codec == "flac" {
		return pb.AudioCodec_FLAC.Enum()
	} else if codec == "ac3" {
		return pb.AudioCodec_AC3.Enum()
	}

	return pb.AudioCodec_COPY_ACODEC.Enum()
}
