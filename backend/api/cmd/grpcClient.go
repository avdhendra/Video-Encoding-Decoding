package main

import (
	"context"
	"time"

	pb "video-encoding/shared/proto/job"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProducerClient struct {
	cc  *grpc.ClientConn
	api pb.JobProducerServiceClient
}

func NewProducerClient(target string) (*ProducerClient, error) {
	cc, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: 3 * time.Second,
		}),
	)
	if err != nil {
		return nil, err
	}
	return &ProducerClient{
		cc:  cc,
		api: pb.NewJobProducerServiceClient(cc),
	}, nil
}

func (c *ProducerClient) Close() error { return c.cc.Close() }

func (c *ProducerClient) Enqueue(ctx context.Context, req *pb.EnqueueTranscodeJobRequest) (*pb.EnqueueTranscodeJobResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.api.EnqueueTranscodeJob(ctx, req)
}
