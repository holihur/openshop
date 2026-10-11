"use client";

import Link from "next/link";
import { useState } from "react";

import type { Locale } from "@/lib/i18n";

export interface Announcement {
  message: string;
  url?: string;
}

/**
 * The banner messages, paged inside one line. Rendered on the client so paging
 * works without a round trip, while the messages themselves came from the
 * server-rendered payload.
 */
export function AnnouncementBar({
  announcements,
  locale,
}: {
  announcements: Announcement[];
  locale: Locale;
}) {
  const [index, setIndex] = useState(0);
  if (announcements.length === 0) return null;
  const current = announcements[Math.min(index, announcements.length - 1)];
  const zh = locale === "zh";
  const total = announcements.length;

  return (
    <div className="flex items-center justify-center gap-3">
      {total > 1 && (
        <button
          type="button"
          onClick={() => setIndex((i) => (i - 1 + total) % total)}
          aria-label={zh ? "上一条公告" : "Previous announcement"}
          className="hover:opacity-70"
        >
          ‹
        </button>
      )}
      <span aria-live="polite">
        {current.url ? (
          <Link href={current.url} className="underline">
            {current.message}
          </Link>
        ) : (
          current.message
        )}
      </span>
      {total > 1 && (
        <>
          <button
            type="button"
            onClick={() => setIndex((i) => (i + 1) % total)}
            aria-label={zh ? "下一条公告" : "Next announcement"}
            className="hover:opacity-70"
          >
            ›
          </button>
          <span className="flex items-center gap-1">
            {announcements.map((announcement, i) => (
              <button
                key={`${announcement.message}-${i}`}
                type="button"
                aria-label={zh ? `第 ${i + 1} 条公告，共 ${total} 条` : `Announcement ${i + 1} of ${total}`}
                onClick={() => setIndex(i)}
                className={`size-1.5 rounded-full ${i === index ? "bg-current" : "bg-current/40"}`}
              />
            ))}
          </span>
        </>
      )}
    </div>
  );
}
