"use client";

import Link from "next/link";
import useSWR from "swr";

import VideoCard from "./VideoCard";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { listVideos } from "../libs/api";

export default function VideoGrid() {
  const { data, isLoading, error } = useSWR(["videos", 24, 0], () => listVideos(24, 0), {
    refreshInterval: 10_000, // refresh list periodically
  });
  console.log("VideoGrid data:", data);
  console.log("VideoGrid error:", error);

  return (
    <div className="space-y-6">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Videos</h1>
          <p className="text-sm text-muted-foreground">Upload, transcode to HLS, and play with adaptive quality.</p>
        </div>
        <Link href="/upload">
          <Button className="rounded-2xl">Upload</Button>
        </Link>
      </header>

      {error && (
        <div className="rounded-xl border p-4 text-sm">
          Failed to load videos. Please try again.
        </div>
      )}

      {isLoading && (
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 md:grid-cols-3">
          {Array.from({ length: 9 }).map((_, i) => (
            <div key={i} className="space-y-3">
              <Skeleton className="h-40 w-full rounded-2xl" />
              <Skeleton className="h-5 w-3/4" />
              <Skeleton className="h-4 w-1/2" />
            </div>
          ))}
        </div>
      )}

      {data && (
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 md:grid-cols-3">
          {data.items.map((v) => (
            <VideoCard key={v.id} video={v} />
          ))}
        </div>
      )}
    </div>
  );
}
