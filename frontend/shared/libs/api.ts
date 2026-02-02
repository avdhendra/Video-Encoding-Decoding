import axios from "axios";
import type { PlaybackResp, PresignReq, PresignResp, VideoDetail, VideoListItem } from "./types";

const baseURL = process.env.NEXT_PUBLIC_API_BASE_URL;

if (!baseURL) throw new Error("Missing NEXT_PUBLIC_API_BASE_URL");

export const api = axios.create({
  baseURL,
  timeout: 20000,
});

export async function listVideos(limit = 24, offset = 0) {
  const { data } = await api.get("/videos", { params: { limit, offset } });
  return data.data as { items: VideoListItem[]; total: number; limit: number; offset: number };
}

export async function getVideo(id: string) {
  const { data } = await api.get(`/videos/${id}`);
  return data.data as VideoDetail;
}

export async function presignUpload(payload: PresignReq) {
  const { data } = await api.post("/videos/presign", payload);
  return data.data as PresignResp;
}

export async function startJob(videoId: string, pipeline: string = "hls") {
  const { data } = await api.post(`/videos/${videoId}/jobs`, { pipeline });
  return data.data as { videoId: string; jobId: string; status: string };
}

export async function getPlayback(videoId: string) {
  const { data } = await api.get(`/videos/${videoId}/playback`);
  return data.data as PlaybackResp;
}

// Direct-to-S3 PUT
export async function putFileToPresignedUrl(url: string, file: File, contentType?: string) {
  await axios.put(url, file, {
    headers: {
      "Content-Type": contentType || file.type || "application/octet-stream",
    },
    // Large files: prevent axios from choking
    maxBodyLength: Infinity,
    maxContentLength: Infinity,
  });
}
