import { Outlet } from "react-router-dom";

import { Footer } from "@/components/layout/Footer";
import { Header } from "@/components/layout/Header";
import { EmailVerificationBanner } from "@/components/email-verification-banner";

export function AppLayout() {
  return (
    <div className="flex min-h-dvh flex-col">
      <Header />
      <EmailVerificationBanner />
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-8">
        <Outlet />
      </main>
      <Footer />
    </div>
  );
}
