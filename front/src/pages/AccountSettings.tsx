import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Download, KeyRound, Trash2 } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { api } from "@lib/api";
import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";

export function AccountSettingsPage() {
  const { user, logout } = useAuth();
  const { t } = useI18n();
  const navigate = useNavigate();

  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [busy, setBusy] = useState(false);

  async function changePassword(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api.post("/auth/password/change", { currentPassword: current, newPassword: next });
      toast.success(t("account.passwordChanged"));
      await logout();
      navigate("/login", { replace: true });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("account.changeFailed"));
    } finally {
      setBusy(false);
    }
  }

  async function exportData() {
    try {
      const data = await api.get<unknown>("/auth/me/export");
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "openshop-export.json";
      a.click();
      URL.revokeObjectURL(url);
      toast.success(t("account.exported"));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("account.exportFailed"));
    }
  }

  async function deleteAccount() {
    if (!window.confirm(t("account.deleteConfirm"))) {
      return;
    }
    try {
      await api.del("/auth/me");
      await logout();
      toast.success(t("account.deleted"));
      navigate("/", { replace: true });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("account.deletionFailed"));
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="text-2xl font-bold">{t("account.settings")}</h1>

      <Card>
        <CardHeader>
          <CardTitle>{t("account.changePassword")}</CardTitle>
          <CardDescription>{t("account.changePasswordSubtitle")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={changePassword} className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="current">{t("auth.currentPassword")}</Label>
              <Input
                id="current"
                type="password"
                value={current}
                onChange={(e) => setCurrent(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="next">{t("auth.newPassword")}</Label>
              <Input
                id="next"
                type="password"
                value={next}
                onChange={(e) => setNext(e.target.value)}
                minLength={8}
                required
              />
            </div>
            <div className="sm:col-span-2">
              <Button type="submit" disabled={busy}>
                <KeyRound className="size-4" />
                {busy ? t("common.saving") : t("account.changePassword")}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("account.yourData")}</CardTitle>
          <CardDescription>{t("account.exportSubtitle")}</CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="outline" onClick={exportData}>
            <Download className="size-4" />
            {t("account.export")}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">{t("account.deleteTitle")}</CardTitle>
          <CardDescription>{t("account.deleteSubtitle")}</CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="destructive" onClick={deleteAccount} disabled={!user}>
            <Trash2 className="size-4" />
            {t("account.delete")}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
