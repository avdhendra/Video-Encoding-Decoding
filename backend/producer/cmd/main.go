package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	kafka "video-encoding/producer/internal"
	
	"video-encoding/shared/env"
	

	"google.golang.org/grpc"
	grpcserver "google.golang.org/grpc"
)

type config struct {
	broker  string
	groupId string
	topic   string
	grpcAddr string
}



type application struct {
	config config
	
	
}

func main() {

	cfg := config{
		broker: env.GetString("BROKER", "kafka:9092"),
		groupId: env.GetString("GROUPID","consumer-group-1"),
		topic: env.GetString("TOPIC","customers"),
		grpcAddr: env.GetString("GRPC_ADDR", ":50051"),
	}
    
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	lis, err := net.Listen("tcp", cfg.grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	p,err:= kafka.NewProducer(cfg.broker)
	if err != nil {
		log.Fatalf("Kafka producer failed: %v", err)
	}
	defer p.Close()

	grpcServer := grpcserver.NewServer(grpc.ConnectionTimeout(10*time.Second),)
	NewGrpcHandler(grpcServer,p,cfg.topic)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()

	// wait for the shutdown signal
	<-ctx.Done()
	log.Println("Shutting down the server...")
	grpcServer.GracefulStop()
}
