"use client";

import { useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import useSWR from "swr";

import { Skeleton } from "@/components/ui/skeleton";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { getPlayback, getVideo } from "@/shared/libs/api";
import PlaybackPanel from "@/shared/components/PlaybackPanel";
import VideoPlayer from "@/shared/components/VideoPlayer";

export default function WatchPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;

  const videoQ = useSWR(["video", id], () => getVideo(id));
  const playbackQ = useSWR(["playback", id], () => getPlayback(id), {
    refreshInterval: (data) => (data?.playbackReady ? 15000 : 1500),
  });

  const pb = playbackQ.data;

  // Stable playback URL (safe for render)
  const [masterUrlStable, setMasterUrlStable] = useState<string | undefined>();

  // Internal guard so we only set it once
  const lockedRef = useRef(false);

  useEffect(() => {
    if (!lockedRef.current && pb?.playbackReady && pb?.masterUrl) {
      lockedRef.current = true;
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setMasterUrlStable(pb.masterUrl);
    }
  }, [pb?.playbackReady, pb?.masterUrl]);

  if (videoQ.isLoading) {
    return (
      <main className="mx-auto max-w-6xl px-4 py-6 space-y-6">
        <Skeleton className="h-10 w-40" />
        <Skeleton className="aspect-video w-full rounded-2xl" />
      </main>
    );
  }

  if (videoQ.error) {
    return (
      <main className="mx-auto max-w-6xl px-4 py-6">
        <div className="rounded-xl border p-4">Failed to load video.</div>
      </main>
    );
  }

  const video = videoQ.data!;

  return (
    <main className="mx-auto max-w-6xl px-4 py-6 space-y-6">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">{video.title || "Untitled"}</h1>
          <p className="text-sm text-muted-foreground">{video.description || "—"}</p>
        </div>
        <Link href="/">
          <Button variant="outline" className="rounded-2xl">
            Back
          </Button>
        </Link>
      </header>

      <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
        <VideoPlayer
          masterUrl={masterUrlStable}
          availableRenditions={pb?.availableRenditions || []}
          playbackReady={pb?.playbackReady || false}
        />

        <PlaybackPanel playback={pb} />
      </div>
    </main>
  );
}
