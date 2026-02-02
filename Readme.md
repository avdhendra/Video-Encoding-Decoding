# 🎬 Video Encoding & Adaptive Streaming Platform

A production-grade video upload, transcoding, and adaptive streaming system inspired by YouTube.

Built with Go microservices, Kafka (Confluent), PostgreSQL, AWS S3, FFmpeg, and a modern Next.js + shadcn/ui frontend.

## ✨ Features

* 📤 Direct video + thumbnail upload to S3 using presigned URLs

* ⚙ Background transcoding pipeline via Kafka

* 📡 HLS adaptive streaming (480p / 720p / 1080p)

* 📊 Real-time progress updates

* 📉 Automatic quality switching based on network conditions

* 🎛 Manual quality selector (YouTube-style)

* 🧩 Microservice-ready architecture
```bash 
🧱 Architecture
Frontend (Next.js)
     |
     v
API Service (Go)
     |
     v
Producer Service (Kafka publisher)
     |
     v
Kafka (Confluent)
     |
     v
Worker / Consumer Service (Go + FFmpeg)
     |
     v
AWS S3 (inputs, thumbnails, HLS outputs)
     |
     v
Playback via HLS
```

## 🛠 Tech Stack
### Backend
```
Go

Chi router

PostgreSQL

Kafka (Confluent)

AWS S3 SDK v2

FFmpeg

HLS streaming

Frontend

Next.js (App Router)

Tailwind CSS

shadcn/ui

SWR

hls.js
```


## 🔄 System Flow
### 1️⃣ Upload

* Frontend requests presigned URLs:
```
POST /v1/videos/presign
```

* Uploads directly to S3.

### 2️⃣ Create Job
```
POST /v1/videos/{id}/jobs
```

* API calls producer → publishes Kafka job message.

### 3️⃣ Transcoding Worker

```
Consumer:

downloads video

runs FFmpeg

generates HLS renditions

uploads outputs

updates DB
```
### 4️⃣ Playback

```
Frontend polls:

GET /v1/videos/{id}/playback

```
### Receives signed master playlist URL.

### 📡 Adaptive Streaming

Uses:
```
HLS protocol

hls.js ABR engine

Supports:

automatic bitrate switching

manual quality override

Just like YouTube.
```


### 🐳 Running with Docker
Prerequisites
```
Docker

Docker Compose

AWS S3 bucket

▶ Start all services
docker-compose up --build
```
### 🌐 Services
* Service	Port

```Frontend	3000
API	8080
Postgres	5432
Kafka	9092
```
### 🔐 Environment Variables

```API
DB_ADDR=postgres://...
S3_BUCKET=your-bucket
S3_REGION=eu-central-1
BROKER=kafka:9092

Producer
BROKER=kafka:9092
TOPIC=video.transcode.jobs

Consumer
BROKER=kafka:9092
DB_ADDR=postgres://...

```