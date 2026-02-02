"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Progress } from "@/components/ui/progress";

import Link from "next/link";
import { toast } from "sonner";
import { presignUpload, putFileToPresignedUrl, startJob } from "@/shared/libs/api";

type Phase = "idle" | "presigning" | "uploading" | "starting" | "done";

export default function UploadPage() {
 
  const [phase, setPhase] = useState<Phase>("idle");
  const [pct, setPct] = useState(0);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [videoFile, setVideoFile] = useState<File | null>(null);
  const [thumbFile, setThumbFile] = useState<File | null>(null);

  async function onSubmit() {
    if (!videoFile) {
      toast.error("Video file required");
      return;
    }
    if (!thumbFile) {
      toast.error("Thumbnail required");
      return;
    }

    try {
      setPhase("presigning");
      setPct(5);

      const presign = await presignUpload({
        title,
        description,
        videoFilename: videoFile.name,
        videoType: videoFile.type || "video/mp4",
        thumbFilename: thumbFile.name,
        thumbType: thumbFile.type || "image/jpeg",
      });

      setPhase("uploading");
      setPct(15);

      // Upload thumbnail first (fast feedback)
      await putFileToPresignedUrl(presign.thumbPutUrl, thumbFile, thumbFile.type);
      setPct(40);

      // Upload video (can take time)
      await putFileToPresignedUrl(presign.videoPutUrl, videoFile, videoFile.type);
      setPct(70);

      setPhase("starting");
      await startJob(presign.videoId, "hls");
      setPct(95);

      setPhase("done");
      setPct(100);

      toast.success("Upload started");
      window.location.href = `/watch/${presign.videoId}`;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) {
      toast.error( "Upload failed");
      setPhase("idle");
      setPct(0);
    }
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Upload</h1>
          <p className="text-sm text-muted-foreground">Upload a video + thumbnail, then we generate HLS renditions.</p>
        </div>
        <Link href="/">
          <Button variant="outline" className="rounded-2xl">Back</Button>
        </Link>
      </div>

      <Card className="rounded-2xl">
        <CardHeader className="space-y-1">
          <div className="text-sm text-muted-foreground">
            {phase === "idle" && "Fill details and upload files."}
            {phase === "presigning" && "Preparing upload…"}
            {phase === "uploading" && "Uploading to storage…"}
            {phase === "starting" && "Starting transcoding…"}
            {phase === "done" && "Done."}
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          <div className="grid gap-2">
            <label className="text-sm font-medium">Title</label>
            <Input className="rounded-2xl" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="My video" />
          </div>

          <div className="grid gap-2">
            <label className="text-sm font-medium">Description</label>
            <Textarea className="rounded-2xl" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="What is this video about?" />
          </div>

          <div className="grid gap-2">
            <label className="text-sm font-medium">Thumbnail</label>
            <Input className="rounded-2xl" type="file" accept="image/*" onChange={(e) => setThumbFile(e.target.files?.[0] ?? null)} />
          </div>

          <div className="grid gap-2">
            <label className="text-sm font-medium">Video</label>
            <Input className="rounded-2xl" type="file" accept="video/*" onChange={(e) => setVideoFile(e.target.files?.[0] ?? null)} />
          </div>

          {(phase !== "idle") && (
            <div className="space-y-2">
              <Progress value={pct} />
              <div className="text-xs text-muted-foreground">{pct}%</div>
            </div>
          )}

          <Button
            className="w-full rounded-2xl"
            onClick={onSubmit}
            disabled={phase !== "idle"}
          >
            Upload & Transcode
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
