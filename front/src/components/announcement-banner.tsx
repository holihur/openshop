import { useState } from "react";
import { Link } from "react-router-dom";
import { Megaphone, X } from "lucide-react";

import { useSite } from "@lib/hooks/useSite";
import { useI18n } from "@lib/i18n";

// Site-wide announcement, editable from the ops console. Dismissal is keyed by
// the message, so publishing a new one re-shows the banner.
const DISMISS_KEY = "openshop.announcementDismissed";

export function AnnouncementBanner() {
  const { data: site } = useSite();
  const { t } = useI18n();
  const message = site?.announcement?.message?.trim() ?? "";
  const [dismissed, setDismissed] = useState(() => localStorage.getItem(DISMISS_KEY) ?? "");

  if (!message || dismissed === message) return null;

  const url = site?.announcement?.url?.trim();

  function dismiss() {
    localStorage.setItem(DISMISS_KEY, message);
    setDismissed(message);
  }

  return (
    <div className="bg-primary text-primary-foreground">
      <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-2 text-sm">
        <Megaphone className="size-4 shrink-0" aria-hidden />
        {url ? (
          <Link to={url} className="flex-1 underline-offset-2 hover:underline">
            {message}
          </Link>
        ) : (
          <span className="flex-1">{message}</span>
        )}
        <button
          type="button"
          onClick={dismiss}
          aria-label={t("common.dismiss")}
          className="shrink-0 opacity-80 hover:opacity-100"
        >
          <X className="size-4" />
        </button>
      </div>
    </div>
  );
}
