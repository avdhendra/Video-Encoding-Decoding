"use client";

import Link from "next/link";
import Image from "next/image";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { VideoListItem } from "../libs/types";


function statusVariant(status: VideoListItem["status"]) {
  if (status === "ready") return "default";
  if (status === "failed") return "destructive";
  if (status === "processing") return "secondary";
  return "outline";
}

export default function VideoCard({ video }: { video: VideoListItem }) {
  return (
    <Link href={`/watch/${video.id}`} className="group">
      <Card className="overflow-hidden rounded-2xl transition-shadow hover:shadow-md">
        <div className="relative aspect-video bg-muted">
          {video.thumbnailUrl ? (
            <Image
              src={video.thumbnailUrl}
              alt={video.title || "Thumbnail"}
              fill
              className="object-cover"
              unoptimized
            />
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              No thumbnail
            </div>
          )}

          <div className="absolute left-3 top-3">
            <Badge variant={statusVariant(video.status)} className="rounded-xl">
              {video.status}
            </Badge>
          </div>
        </div>

        <CardContent className="space-y-1 p-4">
          <div className="line-clamp-2 font-medium leading-snug group-hover:underline">
            {video.title || "Untitled"}
          </div>
          <div className="line-clamp-1 text-sm text-muted-foreground">
            {video.description || "—"}
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
