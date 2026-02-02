"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Hls from "hls.js";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import QualitySelector, { QualityMode } from "./QualitySelector";

type Props = {
  masterUrl?: string;
  availableRenditions: string[];
  playbackReady: boolean;
};

export default function VideoPlayer({ masterUrl, availableRenditions, playbackReady }: Props) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const hlsRef = useRef<Hls | null>(null);

  const [mode, setMode] = useState<QualityMode>({ type: "auto" });

  const hasSource = !!masterUrl && playbackReady;

  const qualities = useMemo(() => {
    const order = ["1080p", "720p", "480p"];
    return order.filter((q) => availableRenditions.includes(q));
  }, [availableRenditions]);

  // Init HLS ONCE when source becomes available (or when masterUrl first set)
  useEffect(() => {
    if (!hasSource) return;

    const video = videoRef.current;
    if (!video) return;

    // Prevent re-init if we already have an hls instance
    if (hlsRef.current) return;

    // Safari native HLS
    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = masterUrl!;
      return;
    }

    if (!Hls.isSupported()) return;

    const hls = new Hls({
      enableWorker: true,
      lowLatencyMode: false,
    });

    hlsRef.current = hls;

    hls.on(Hls.Events.ERROR, (_evt, data) => {
      if (!data.fatal) return;

      // Recoverable handling
      if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
        hls.startLoad();
        return;
      }
      if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
        hls.recoverMediaError();
        return;
      }

      // otherwise rebuild
      hls.destroy();
      hlsRef.current = null;
    });

    hls.loadSource(masterUrl!);
    hls.attachMedia(video);

    return () => {
      hls.destroy();
      hlsRef.current = null;
    };
  }, [hasSource, masterUrl]);

  // ABR vs Manual
  useEffect(() => {
    const hls = hlsRef.current;
    if (!hls) return;

    if (mode.type === "auto") {
      hls.currentLevel = -1; // ABR
      return;
    }

    const target = mode.height;
    const levels = hls.levels;

    const idx = levels.findIndex((l) => l.height === target);
    if (idx >= 0) {
      hls.currentLevel = idx;
      return;
    }

    // fallback closest
    let best = 0;
    let bestDiff = Infinity;
    levels.forEach((l, i) => {
      if (!l.height) return;
      const d = Math.abs(l.height - target);
      if (d < bestDiff) {
        bestDiff = d;
        best = i;
      }
    });
    hls.currentLevel = best;
  }, [mode]);

  return (
    <Card className="rounded-2xl">
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center justify-between">
          <div className="text-sm font-medium">Player</div>
          <QualitySelector
            availableRenditions={qualities}
            value={mode}
            onChange={setMode}
            disabled={!hasSource}
          />
        </div>

        <div className="relative aspect-video overflow-hidden rounded-2xl bg-black">
          {!hasSource ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="w-[80%] space-y-3">
                <Skeleton className="h-6 w-2/3 mx-auto" />
                <Skeleton className="h-4 w-1/2 mx-auto" />
              </div>
            </div>
          ) : (
            <video ref={videoRef} className="h-full w-full" controls playsInline />
          )}
        </div>

        <p className="text-xs text-muted-foreground">
          Keep <b>Auto</b> enabled for YouTube-like switching when bandwidth drops (4G → 3G).
        </p>
      </CardContent>
    </Card>
  );
}
