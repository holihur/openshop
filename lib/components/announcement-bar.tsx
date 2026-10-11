import { useEffect, useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";

import { useI18n } from "@lib/i18n";

export interface Announcement {
  message: string;
  url?: string;
}

/**
 * The site-wide banner. Several messages are paged rather than stacked, so the
 * bar keeps one line of height no matter how much the operator has to say; a
 * single message renders without any controls.
 */
export function AnnouncementBar({
  announcements,
  className,
  renderMessage,
  onIndexChange,
}: {
  announcements: Announcement[];
  className?: string;
  /** Lets the host render the message (the SPA uses router links). */
  renderMessage?: (announcement: Announcement) => React.ReactNode;
  onIndexChange?: (index: number) => void;
}) {
  const { t } = useI18n();
  const [index, setIndex] = useState(0);
  const total = announcements.length;

  // A shorter list (an operator edit reaching the client) must not point past
  // its end.
  useEffect(() => {
    if (index >= total) {
      setIndex(0);
      onIndexChange?.(0);
    }
  }, [index, total, onIndexChange]);

  if (total === 0) return null;

  const current = announcements[Math.min(index, total - 1)];
  const move = (delta: number) => {
    const next = (index + delta + total) % total;
    setIndex(next);
    onIndexChange?.(next);
  };

  return (
    <div className={className ?? "bg-muted px-4 py-2 text-center text-sm"}>
      <div className="mx-auto flex max-w-6xl items-center justify-center gap-3">
        {total > 1 && (
          <button
            type="button"
            onClick={() => move(-1)}
            aria-label={t("announcement.previous")}
            className="hover:opacity-70"
          >
            <ChevronLeft className="size-4" />
          </button>
        )}

        <p aria-live="polite" className="min-w-0">
          {renderMessage
            ? renderMessage(current)
            : current.url
              ? (
                  <a href={current.url} className="underline">
                    {current.message}
                  </a>
                )
              : current.message}
        </p>

        {total > 1 && (
          <>
            <button
              type="button"
              onClick={() => move(1)}
              aria-label={t("announcement.next")}
              className="hover:opacity-70"
            >
              <ChevronRight className="size-4" />
            </button>
            <span className="text-muted-foreground flex items-center gap-1" role="tablist">
              {announcements.map((announcement, i) => (
                <button
                  key={`${announcement.message}-${i}`}
                  type="button"
                  role="tab"
                  aria-selected={i === index}
                  aria-label={t("announcement.goTo", { index: i + 1, total })}
                  onClick={() => {
                    setIndex(i);
                    onIndexChange?.(i);
                  }}
                  className={
                    i === index
                      ? "bg-foreground size-1.5 rounded-full"
                      : "bg-muted-foreground/40 size-1.5 rounded-full"
                  }
                />
              ))}
            </span>
          </>
        )}
      </div>
    </div>
  );
}
