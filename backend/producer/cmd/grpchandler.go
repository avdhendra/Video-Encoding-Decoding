package main

import (
	"context"
	"time"
	kafka "video-encoding/producer/internal"
	"video-encoding/shared/types"

	pb "video-encoding/shared/proto/job" 

	"google.golang.org/grpc"
)

type grpcHandler struct {
	
    pb.UnimplementedJobProducerServiceServer
	producer *kafka.Producer
	topic    string
}

func NewGrpcHandler(s *grpc.Server,prod *kafka.Producer, topic string) {
	handler := &grpcHandler{
		producer: prod,
		topic:    topic,
	}

	pb.RegisterJobProducerServiceServer(s, handler)
	
	
}


func (s *grpcHandler) EnqueueTranscodeJob(ctx context.Context, req *pb.EnqueueTranscodeJobRequest) (*pb.EnqueueTranscodeJobResponse, error) {
	if req.GetJobId() == "" || req.GetVideoId() == "" || req.GetInputKey() == "" {
		return &pb.EnqueueTranscodeJobResponse{
			Accepted: false,
			Message:  "job_id, video_id, input_key are required",
		}, nil
	}

	pipeline := req.GetPipeline()
	if pipeline == "" {
		pipeline = "hls"
	}

	// publish to kafka
	msg := types.TranscodeJobMessage{
		JobID:    req.GetJobId(),
		VideoID:  req.GetVideoId(),
		InputKey: req.GetInputKey(),
		Pipeline: pipeline,
	}

	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.producer.PublishJSON(pctx, s.topic, msg.JobID, msg); err != nil {
		
		return &pb.EnqueueTranscodeJobResponse{
			Accepted: false,
			Message:  "kafka publish failed: " + err.Error(),
		}, nil
	}

	return &pb.EnqueueTranscodeJobResponse{
		Accepted: true,
		Message:  "enqueued",
	}, nil
}