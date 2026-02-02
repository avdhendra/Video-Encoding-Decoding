"use client";


import { Card, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { PlaybackResp } from "../libs/types";

export default function PlaybackPanel({ playback }: { playback?: PlaybackResp }) {
  if (!playback) {
    return (
      <Card className="rounded-2xl">
        <CardContent className="p-4 space-y-3">
          <Skeleton className="h-5 w-1/2" />
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-3 w-full" />
        </CardContent>
      </Card>
    );
  }

  const status = playback.status || "unknown";
  const ready = !!playback.playbackReady;

  return (
    <Card className="rounded-2xl">
      <CardContent className="p-4 space-y-4">
        <div className="flex items-center justify-between">
          <div className="text-sm font-medium">Transcoding</div>
          <Badge variant={ready ? "default" : "secondary"} className="rounded-xl">
            {ready ? "Playback Ready" : status}
          </Badge>
        </div>

        <div className="space-y-2">
          <Progress value={playback.progress ?? 0} />
          <div className="text-xs text-muted-foreground">
            {playback.progress ?? 0}% • Renditions: {(playback.availableRenditions || []).join(", ") || "—"}
          </div>
        </div>

        {!ready && (
          <div className="text-sm text-muted-foreground">
            Your video is being processed. You can stay on this page — it will become playable automatically.
          </div>
        )}

        {ready && playback.masterUrl && (
          <div className="text-xs text-muted-foreground break-all">
            Master: {playback.masterUrl}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
