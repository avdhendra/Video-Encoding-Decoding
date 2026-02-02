"use client";

import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";

export type QualityMode =
  | { type: "auto" }
  | { type: "fixed"; height: number; label: string };

const map = (q: string) => {
  if (q === "1080p") return { height: 1080, label: "1080p" };
  if (q === "720p") return { height: 720, label: "720p" };
  return { height: 480, label: "480p" };
};

export default function QualitySelector({
  availableRenditions,
  value,
  onChange,
  disabled,
}: {
  availableRenditions: string[];
  value: QualityMode;
  onChange: (v: QualityMode) => void;
  disabled?: boolean;
}) {
  const label = value.type === "auto" ? "Auto" : value.label;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className="rounded-2xl" disabled={disabled}>
          Quality: {label}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="rounded-2xl">
        <DropdownMenuItem onClick={() => onChange({ type: "auto" })}>Auto</DropdownMenuItem>
        {availableRenditions.map((q) => {
          const m = map(q);
          return (
            <DropdownMenuItem
              key={q}
              onClick={() => onChange({ type: "fixed", height: m.height, label: m.label })}
            >
              {q}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
