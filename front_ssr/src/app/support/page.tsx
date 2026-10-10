import Link from "next/link";

import { apiList } from "@/lib/api";
import { AccountNav } from "@/components/account-nav";
import { TicketForm } from "@/components/ticket-form";
import { formatDate } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { Ticket } from "@/lib/types";

/**
 * Support. A guest may open a ticket; a signed-in shopper also sees the ones
 * they already have, because the list endpoint needs a session.
 */
export default async function SupportPage() {
  const locale = await resolveLocale();
  const t = translator(locale);
  const shopper = await getShopper();
  const tickets = shopper.token
    ? await apiList<Ticket>("/tickets?pageSize=20", { token: shopper.token }).catch(
        () => ({ items: [] as Ticket[], total: 0, page: 1, pageSize: 20 }),
      )
    : { items: [] as Ticket[], total: 0, page: 1, pageSize: 20 };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("account.support")}</h1>
      {shopper.token ? <AccountNav locale={locale} current="/support" /> : null}

      <TicketForm
        signedIn={Boolean(shopper.token)}
        labels={{
          title: t("support.new"),
          subject: t("support.subject"),
          body: t("support.message"),
          email: t("auth.email"),
          name: t("auth.name"),
          submit: t("support.submit"),
          sending: t("cart.updating"),
          failed: t("error.generic"),
        }}
      />

      {tickets.items.length > 0 ? (
        <section className="space-y-2">
          <h2 className="font-semibold">{t("support.mine")}</h2>
          <ul className="divide-y rounded-lg border">
            {tickets.items.map((ticket) => (
              <li key={ticket.id} className="flex items-center justify-between gap-3 p-3 text-sm">
                <Link href={`/support/${ticket.id}`} className="font-medium underline">
                  {ticket.subject}
                </Link>
                <span className="text-muted-foreground">{ticket.status}</span>
                <span className="text-muted-foreground text-xs">
                  {formatDate(ticket.updatedAt, locale)}
                </span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}
