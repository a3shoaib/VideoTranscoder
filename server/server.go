package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	pb "projects/arshoaib/largefile-streaming/server/exports/compiled_proto"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	pb.UnimplementedVideoTranscoderServiceServer
}

func (s *Server) SendVideoChunk(stream pb.VideoTranscoderService_SendVideoChunkServer) error {
	// Obtain the header from the stream, which is the first message of the stream.
	fmt.Println("checkpoint 0")
	firstChunk, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("Failed to receive the first chunk from the stream: %v", err)
	}
	header := firstChunk.GetHeader()
	if header == nil {
		return fmt.Errorf("Failed to receive header from the stream. The first call to `SendVideoChunk` MUST populate the `header` field : %v", err)
	}

	fmt.Println("checkpoint 1")

	// Validate the file name supplied contains an extension.
	fileExtension := filepath.Ext(header.FileName)
	if len(fileExtension) <= 0 {
		return fmt.Errorf("Must provide an extension in the input file path, otherwise output file format cannot be inferred.")
	}

	fmt.Println("checkpoint 2")
	// Create temporary file to buffer the input file in (input to ffmpeg).
	inputFile, err := os.CreateTemp("/tmp", fmt.Sprintf("input_video*%s", fileExtension))
	if err != nil {
		return fmt.Errorf("Error creating temporary input file: %v", err)
	}
	defer os.Remove(inputFile.Name())
	fmt.Println("checkpoint 3")

	// Create a writer buffer on the temporary file.
	writer := bufio.NewWriter(inputFile)
	for {
		// Receive the next video chunk from the stream.
		videoChunk, err := stream.Recv()

		// If this is the last chunk, break out of the loop.
		if err == io.EOF {
			break
		}

		// If there was an error receiving the chunk, log and fail.
		if err != nil {
			return fmt.Errorf("Failed to receive chunk: %v", err)
		}

		// Write the chunk bytes to the buffer.
		writer.Write(videoChunk.GetChunkData())
	}
	fmt.Println("checkpoint 4")

	// Flush out all remaining data from the buffer into the file.
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Could not flush file to buffer: %v", err)
	}
	fmt.Println("checkpoint 5")

	// Create temporary file to write the output file to (output from ffmpeg).
	outputFile, err := os.CreateTemp("/tmp", fmt.Sprintf("output_video*%s", fileExtension))
	if err != nil {
		return fmt.Errorf("Error creating temporary output file: %v", err)
	}
	defer os.Remove(outputFile.Name())
	fmt.Println("checkpoint 6")

	// Run ffmpeg and output results to the output file.
	ffmpegCommandArgs := []string{"-y", "-i", inputFile.Name(),
		"-vcodec", GetVideoCodecName(header.GetOutputVideoCodec()),
		"-acodec", GetAudioCodecName(header.GetOutputAudioCodec()),
		outputFile.Name()}
	cmd := exec.Command("ffmpeg", ffmpegCommandArgs...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error while executing ffmpeg command: %v", err)
	}
	fmt.Println("checkpoint 7")

	// Create an S3 session.
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(header.AwsCredentials.Region),
		Credentials: credentials.NewStaticCredentials(
			header.AwsCredentials.AccessKeyId, header.AwsCredentials.SecretAccessKey, ""),
	})
	if err != nil {
		return fmt.Errorf("Error creating S3 session: %v", err)
	}
	fmt.Println("checkpoint 8")

	// Create the file name to send to S3, which is "<original_file_name>_transcoded<extension>"
	extensionName := filepath.Ext(header.FileName)
	fileNameWithoutExtension := strings.TrimSuffix(header.FileName, extensionName)
	s3FileName := fmt.Sprintf("%s_transcoded%s", fileNameWithoutExtension, extensionName)

	// Upload the file in parts to S3.
	err = UploadToS3(awsSession, header.AwsCredentials.BucketName, s3FileName, outputFile)
	if err != nil {
		return fmt.Errorf("Error uploading to S3: %v", err)
	}

	stream.SendAndClose(&emptypb.Empty{})
	fmt.Println("checkpoint 9")
	return nil
}

func GetVideoCodecName(codec pb.VideoCodec) string {
	if codec == pb.VideoCodec_H264 {
		return "h264"
	} else if codec == pb.VideoCodec_AV1 {
		return "av1"
	} else if codec == pb.VideoCodec_HEVC {
		return "hevc"
	} else if codec == pb.VideoCodec_VP9 {
		return "vp9"
	}
	return "copy"
}

func GetAudioCodecName(codec pb.AudioCodec) string {
	if codec == pb.AudioCodec_AAC {
		return "aac"
	} else if codec == pb.AudioCodec_AC3 {
		return "av1"
	} else if codec == pb.AudioCodec_FLAC {
		return "flac"
	}
	return "copy"
}

// UploadToS3 uses an upload manager to upload data to an object in a bucket.
// The upload manager breaks large data into parts and uploads the parts concurrently.
func UploadToS3(awsSession *session.Session, bucketName string, fileName string, file *os.File) error {
	var partMiBs int64 = 10

	uploader := s3manager.NewUploader(awsSession, func(u *s3manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
	})
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("Couldn't upload large object to %v:%v. Here's why: %v\n",
			bucketName, fileName, err)
	}

	return err
}
