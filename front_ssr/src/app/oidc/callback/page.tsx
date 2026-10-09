"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

/**
 * Landing page for a provider sign-in. The API hands the tokens back in the URL
 * fragment (which a server never receives), so this page forwards them to a
 * route handler that stores them in httpOnly cookies and then continues to the
 * account area.
 */
export default function OidcCallbackPage() {
  const router = useRouter();
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.hash.replace(/^#/, ""));
    const accessToken = params.get("access_token") ?? "";
    if (!accessToken) {
      setFailed(true);
      return;
    }
    fetch("/api/auth/session", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        accessToken,
        refreshToken: params.get("refresh_token") ?? undefined,
        expiresIn: Number.parseInt(params.get("expires_in") ?? "900", 10) || 900,
      }),
    })
      .then((res) => {
        if (!res.ok) throw new Error("session");
        // Clear the fragment so the tokens do not stay in the address bar.
        window.history.replaceState(null, "", "/account/orders");
        router.replace("/account/orders");
        router.refresh();
      })
      .catch(() => setFailed(true));
  }, [router]);

  return (
    <p className="text-muted-foreground py-16 text-center text-sm">
      {failed ? "Sign in could not be completed." : "Signing you in…"}
    </p>
  );
}
