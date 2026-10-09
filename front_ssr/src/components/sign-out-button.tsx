"use client";

import { useRouter } from "next/navigation";
import { useTransition } from "react";

/** Clears the session cookies and returns to the storefront home page. */
export function SignOutButton({ label }: { label: string }) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();

  async function signOut() {
    await fetch("/api/auth/logout", { method: "POST" });
    startTransition(() => {
      router.push("/");
      router.refresh();
    });
  }

  return (
    <button
      type="button"
      onClick={() => void signOut()}
      disabled={pending}
      className="text-muted-foreground text-sm underline"
    >
      {label}
    </button>
  );
}
