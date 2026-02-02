export type VideoStatus = "uploaded" | "processing" | "ready" | "failed";

export type VideoListItem = {
  id: string;
  title: string;
  description: string;
  thumbnailUrl?: string;
  status: VideoStatus;
  latestJobId?: string | null;
  createdAt: string;
};

export type VideoDetail = {
  id: string;
  title: string;
  description: string;
  thumbnailUrl?: string;
  status: VideoStatus;
  latestJobId?: string | null;
};

export type PresignReq = {
  title: string;
  description: string;
  videoFilename: string;
  videoType?: string;
  thumbFilename: string;
  thumbType?: string;
};

export type PresignResp = {
  videoId: string;
  videoKey: string;
  videoPutUrl: string;
  thumbKey: string;
  thumbPutUrl: string;
};

export type PlaybackResp = {
  videoId: string;
  jobId?: string;
  status: string; // job status
  progress: number;
  playbackReady: boolean;
  availableRenditions?: string[];
  masterKey?: string;
  masterUrl?: string;
};
