import { useState } from "react";
import { MailWarning } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@lib/components/ui/button";
import { api } from "@lib/api";
import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";

// EmailVerificationBanner nudges unverified users to confirm their address.
export function EmailVerificationBanner() {
  const { user } = useAuth();
  const { t } = useI18n();
  const [sent, setSent] = useState(false);
  const [busy, setBusy] = useState(false);

  if (!user || user.emailVerified) return null;
  const email = user.email;

  async function resend() {
    setBusy(true);
    try {
      await api.post("/auth/email/resend", { email });
      setSent(true);
      toast.success(t("banner.sentToast"));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("banner.sendFailed"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="border-b bg-amber-50 text-amber-900 dark:bg-amber-950/40 dark:text-amber-200">
      <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-2 text-sm">
        <MailWarning className="size-4 shrink-0" />
        <span className="flex-1">
          {sent
            ? t("banner.sent")
            : t("banner.verifyPrompt")}
        </span>
        {!sent && (
          <Button variant="outline" size="sm" disabled={busy} onClick={resend}>
            {busy ? t("banner.sending") : t("banner.resend")}
          </Button>
        )}
      </div>
    </div>
  );
}
