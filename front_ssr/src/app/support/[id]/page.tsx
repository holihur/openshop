import Link from "next/link";
import { notFound, redirect } from "next/navigation";

import { apiGet } from "@/lib/api";
import { TicketReply } from "@/components/ticket-reply";
import { formatDate } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { Ticket } from "@/lib/types";

/** One ticket and its conversation. */
export default async function TicketPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const shopper = await getShopper();
  if (!shopper.token) redirect("/login");

  const locale = await resolveLocale();
  const t = translator(locale);
  const ticket = await apiGet<Ticket>(`/tickets/${id}`, { token: shopper.token }).catch(() => null);
  if (!ticket) notFound();

  return (
    <div className="space-y-6">
      <div>
        <Link href="/support" className="text-muted-foreground text-sm underline">
          {t("account.support")}
        </Link>
        <h1 className="text-2xl font-bold">{ticket.subject}</h1>
        <p className="text-muted-foreground text-sm">
          {ticket.reference} · {ticket.status}
        </p>
      </div>

      <ul className="space-y-3">
        {(ticket.messages ?? []).map((message) => (
          <li
            key={message.id}
            className={
              message.authorRole === "customer"
                ? "bg-muted/50 ml-auto max-w-2xl rounded-lg p-3 text-sm"
                : "max-w-2xl rounded-lg border p-3 text-sm"
            }
          >
            <p className="text-muted-foreground text-xs">
              {message.authorName ?? message.authorRole} · {formatDate(message.createdAt, locale)}
            </p>
            <p className="mt-1 whitespace-pre-line">{message.body}</p>
          </li>
        ))}
      </ul>

      <TicketReply
        ticketId={ticket.id}
        labels={{
          title: t("support.reply"),
          placeholder: t("support.message"),
          submit: t("support.send"),
          sending: t("cart.updating"),
          failed: t("error.generic"),
        }}
      />
    </div>
  );
}
