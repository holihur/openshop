import { useState } from "react";
import { Link } from "react-router-dom";
import { Megaphone, X } from "lucide-react";

import { useSite } from "@lib/hooks/useSite";
import { AnnouncementBar } from "@lib/components/announcement-bar";

// Site-wide announcements, editable from the ops console. Several messages are
// paged inside one bar; dismissal is keyed by the visible message, so a new one
// re-shows the banner.
const DISMISS_KEY = "openshop.announcementDismissed";

export function AnnouncementBanner() {
  const { data: site } = useSite();
  // The list is the source of truth; the single setting is a legacy fallback.
  const announcements = (site?.announcements ?? []).filter((item) => item.message?.trim());
  const fallback = site?.announcement?.message?.trim()
    ? [{ message: site.announcement.message, url: site.announcement.url }]
    : [];
  const messages = announcements.length > 0 ? announcements : fallback;

  const [dismissed, setDismissed] = useState(() => localStorage.getItem(DISMISS_KEY) ?? "");
  const [index, setIndex] = useState(0);

  if (messages.length === 0) return null;
  const current = messages[Math.min(index, messages.length - 1)];
  if (dismissed === current.message) return null;

  function dismiss() {
    localStorage.setItem(DISMISS_KEY, current.message);
    setDismissed(current.message);
  }

  return (
    <div className="bg-primary text-primary-foreground">
      <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-2 text-sm">
        <Megaphone className="size-4 shrink-0" aria-hidden />
        <AnnouncementBar
          announcements={messages}
          onIndexChange={setIndex}
          className="min-w-0 flex-1 text-center"
          renderMessage={(announcement) => (
            <AnnouncementLink message={announcement.message} url={announcement.url} />
          )}
        />
        <button
          type="button"
          onClick={dismiss}
          aria-label="Dismiss"
          className="shrink-0 opacity-80 hover:opacity-100"
        >
          <X className="size-4" />
        </button>
      </div>
    </div>
  );
}

/** Link or plain text for one message, used inside the bar. */
export function AnnouncementLink({ message, url }: { message: string; url?: string }) {
  return url ? (
    <Link to={url} className="underline-offset-2 hover:underline">
      {message}
    </Link>
  ) : (
    <span>{message}</span>
  );
}
