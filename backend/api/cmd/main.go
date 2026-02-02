package main

import (
	"context"

	"video-encoding/shared/db"
	"video-encoding/shared/env"
	"video-encoding/shared/store"

	"time"

	"go.uber.org/zap"

	s3_Config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	cfg := config{
		addr:         env.GetString("ADDR", ":8080"),
		apiURL:       env.GetString("EXTERNAL_URL", "localhost:8080"),
		frontendURL:  env.GetString("FRONTEND_URL", "http://localhost:5173"),
		producerGRPC: env.GetString("PRODUCER_GRPC_TARGET", "producer:9095"),

		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://admin:adminpassword@db:5432/REEL_BLOOM?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},

		env: env.GetString("ENV", "development"),

		s3: s3Config{
			region:        env.GetString("S3_REGION", "ap-south-1"),
			bucket:        env.GetString("S3_BUCKET", "your-bucket-name"),
			basePath:      env.GetString("S3_BASE_PATH", "reels/"),
			presignPUTTTL: env.GetDuration("S3_PRESIGN_PUT_TTL", 15*time.Minute),
			presignGETTTL: env.GetDuration("S3_PRESIGN_GET_TTL", 30*time.Minute),
		},
	}

	// Logger
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	// Main Database
	db, err := db.New(
		cfg.db.addr,
		cfg.db.maxOpenConns,
		cfg.db.maxIdleConns,
		cfg.db.maxIdleTime,
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer db.Close()
	logger.Info("database connection pool established")

	awsCfg, err := s3_Config.LoadDefaultConfig(context.Background(), s3_Config.WithRegion(cfg.s3.region))
	if err != nil {
		logger.Fatalw("aws config error", "err", err)
	}
	s3Client := s3.NewFromConfig(awsCfg)
	presigner := s3.NewPresignClient(s3Client)

	store := store.NewStorage(db)

	pc, err := NewProducerClient(cfg.producerGRPC)
	if err != nil {
		logger.Fatalw("producer grpc client init failed", "err", err)
	}
	defer pc.Close()

	app := &application{
		config:    cfg,
		store:     store,
		logger:    logger,
		s3:        s3Client,
		s3Presign: presigner,
		producer: pc,
	}

	mux := app.mount()

	logger.Fatal(app.run(mux))
}
