package main

import (
	"context"
	"time"
	"video-encoding/shared/db"
	"video-encoding/shared/env"
	"video-encoding/shared/store"

	s3_Config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

type config struct {
	broker  string
	groupId string
	topic   string
	db      dbConfig
	s3      s3Config
}

type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type application struct {
	config config
	store  store.Storage
}

type s3Config struct {
	region        string
	bucket        string
	basePath      string
	presignPUTTTL time.Duration
	presignGETTTL time.Duration
}

func main() {
	log := zap.Must(zap.NewProduction()).Sugar()
	defer log.Sync()
	cfg := config{
		broker:  env.GetString("BROKER", "kafka:9092"),
		groupId: env.GetString("GROUPID", "consumer-group-1"),
		topic:   env.GetString("TOPIC", "video.transcode.jobs"),
		s3: s3Config{
			region:        env.GetString("S3_REGION", "ap-south-1"),
			bucket:        env.GetString("S3_BUCKET", "your-bucket-name"),
			basePath:      env.GetString("S3_BASE_PATH", "reels/"),
			presignPUTTTL: env.GetDuration("S3_PRESIGN_PUT_TTL", 15*time.Minute),
			presignGETTTL: env.GetDuration("S3_PRESIGN_GET_TTL", 30*time.Minute),
		},

		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://admin:adminpassword@postgres:5432/cms?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
	}

	db, err := db.New(cfg.db.addr, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)

	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	awsCfg, err := s3_Config.LoadDefaultConfig(context.Background(), s3_Config.WithRegion(cfg.s3.region))
	if err != nil {
		log.Fatalw("aws config error", "err", err)
	}
	s3Client := s3.NewFromConfig(awsCfg)
	store := store.NewStorage(db)
	w := NewWorker(log, store, cfg.broker,
		cfg.groupId,
		cfg.topic,
		s3Client,
		cfg.s3.bucket,
		cfg.s3.basePath)

	ctx := context.Background()
	log.Infow("consumer starting", "brokers", cfg.broker, "topic", cfg.topic, "group", cfg.groupId)
	if err := w.Run(ctx, 250*time.Millisecond); err != nil {
		log.Fatalw("consumer stopped with error", "err", err)
	}
}
