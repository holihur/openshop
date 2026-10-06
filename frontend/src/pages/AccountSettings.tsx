import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Download, KeyRound, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

export function AccountSettingsPage() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [busy, setBusy] = useState(false);

  async function changePassword(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api.post("/auth/password/change", { currentPassword: current, newPassword: next });
      toast.success("Password changed. Please sign in again.");
      await logout();
      navigate("/login", { replace: true });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not change password");
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
      toast.success("Export downloaded");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Export failed");
    }
  }

  async function deleteAccount() {
    if (!window.confirm("Delete your account? This erases your personal data and cannot be undone.")) {
      return;
    }
    try {
      await api.del("/auth/me");
      await logout();
      toast.success("Your account has been deleted");
      navigate("/", { replace: true });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Deletion failed");
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="text-2xl font-bold">Account settings</h1>

      <Card>
        <CardHeader>
          <CardTitle>Change password</CardTitle>
          <CardDescription>You will be signed out of all devices.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={changePassword} className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="current">Current password</Label>
              <Input
                id="current"
                type="password"
                value={current}
                onChange={(e) => setCurrent(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="next">New password</Label>
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
                {busy ? "Saving…" : "Change password"}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Your data</CardTitle>
          <CardDescription>Download a copy of your personal data.</CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="outline" onClick={exportData}>
            <Download className="size-4" />
            Export my data
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">Delete account</CardTitle>
          <CardDescription>
            Erases your profile, addresses, wishlist and reviews. Order records are kept but
            anonymised.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="destructive" onClick={deleteAccount} disabled={!user}>
            <Trash2 className="size-4" />
            Delete my account
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
