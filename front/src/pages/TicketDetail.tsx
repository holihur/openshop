import { useState, type FormEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Skeleton } from "@lib/components/ui/skeleton";
import { Textarea } from "@lib/components/ui/textarea";
import { useMyTicket, useReplyTicket } from "@lib/hooks/useTickets";
import { useI18n } from "@lib/i18n";
import { formatDate } from "@lib/format";
import { cn } from "@lib/utils";
import { TicketStatusBadge } from "./Support";

export function TicketDetailPage() {
  const { id = "" } = useParams();
  const { t } = useI18n();
  const { data, isLoading } = useMyTicket(id);
  const reply = useReplyTicket(id);
  const [body, setBody] = useState("");

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    reply.mutate({ body }, { onSuccess: () => setBody("") });
  }

  if (isLoading) {
    return (
      <div className="mx-auto max-w-3xl py-8">
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }
  if (!data) {
    return (
      <div className="mx-auto max-w-3xl py-8">
        <p className="text-muted-foreground">{t("support.notFound")}</p>
      </div>
    );
  }

  const { ticket, messages } = data;
  const closed = ticket.status === "closed";

  return (
    <div className="mx-auto max-w-3xl space-y-4 py-8">
      <Link to="/support" className="text-muted-foreground inline-flex items-center gap-1 text-sm">
        <ArrowLeft className="size-4" />
        {t("support.back")}
      </Link>
      <div>
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground text-xs">{ticket.reference}</span>
          <TicketStatusBadge status={ticket.status} />
        </div>
        <h1 className="text-xl font-semibold">{ticket.subject}</h1>
      </div>

      <div className="space-y-3">
        {messages.map((m) => (
          <div
            key={m.id}
            className={cn(
              "rounded-lg border p-3",
              m.authorRole === "customer" ? "bg-muted" : "bg-card",
            )}
          >
            <div className="text-muted-foreground mb-1 flex items-center justify-between text-xs">
              <span>
                {m.authorRole === "customer" ? t("support.you") : m.authorName || t("support.staff")}
              </span>
              <span>{formatDate(m.createdAt)}</span>
            </div>
            <p className="text-sm whitespace-pre-wrap">{m.body}</p>
          </div>
        ))}
      </div>

      {closed ? (
        <p className="text-muted-foreground text-sm">{t("support.closedHint")}</p>
      ) : (
        <form onSubmit={onSubmit} className="space-y-2">
          <Textarea
            rows={3}
            required
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder={t("support.replyPlaceholder")}
          />
          <Button type="submit" disabled={reply.isPending}>
            {t("support.reply")}
          </Button>
        </form>
      )}
    </div>
  );
}
