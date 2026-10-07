import { useState, type FormEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Skeleton } from "@lib/components/ui/skeleton";
import { Textarea } from "@lib/components/ui/textarea";
import {
  useAdminTicket,
  useAssignTicketToMe,
  useReplyTicket,
  useUpdateTicket,
} from "@lib/hooks/useTickets";
import { useI18n } from "@lib/i18n";
import { formatDate } from "@lib/format";
import { cn } from "@lib/utils";
import type { TicketKind, TicketPriority, TicketStatus } from "@lib/types";

const STATUSES: TicketStatus[] = ["open", "pending", "resolved", "closed"];
const PRIORITIES: TicketPriority[] = ["low", "normal", "high", "urgent"];
const KINDS: TicketKind[] = ["presale", "postsale", "other"];

export function TicketDetailPage() {
  const { id = "" } = useParams();
  const { t } = useI18n();
  const { data, isLoading } = useAdminTicket(id);
  const reply = useReplyTicket(id, true);
  const update = useUpdateTicket(id);
  const assign = useAssignTicketToMe(id);
  const [body, setBody] = useState("");
  const [internal, setInternal] = useState(false);

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    reply.mutate({ body, internal }, { onSuccess: () => setBody("") });
  }

  if (isLoading) {
    return <Skeleton className="h-96 w-full" />;
  }
  if (!data) {
    return <p className="text-muted-foreground">{t("support.notFound")}</p>;
  }
  const { ticket, messages } = data;

  return (
    <div className="space-y-4">
      <Link to="/tickets" className="text-muted-foreground inline-flex items-center gap-1 text-sm">
        <ArrowLeft className="size-4" />
        {t("ops.tickets")}
      </Link>

      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="text-muted-foreground flex items-center gap-2 text-xs">
            <span>{ticket.reference}</span>
            <span>·</span>
            <span>{ticket.email}</span>
          </div>
          <h1 className="text-xl font-semibold">{ticket.subject}</h1>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={assign.isPending}
          onClick={() => assign.mutate()}
        >
          {t("ops.assignToMe")}
        </Button>
      </div>

      <div className="flex flex-wrap gap-2">
        <select
          aria-label={t("ops.ticketStatus")}
          value={ticket.status}
          onChange={(e) => update.mutate({ status: e.target.value as TicketStatus })}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {t(`ticket.status.${s}`)}
            </option>
          ))}
        </select>
        <select
          aria-label={t("ops.ticketPriority")}
          value={ticket.priority}
          onChange={(e) => update.mutate({ priority: e.target.value as TicketPriority })}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          {PRIORITIES.map((p) => (
            <option key={p} value={p}>
              {t(`ticket.priority.${p}`)}
            </option>
          ))}
        </select>
        <select
          aria-label={t("ops.ticketKind")}
          value={ticket.kind}
          onChange={(e) => update.mutate({ kind: e.target.value as TicketKind })}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          {KINDS.map((k) => (
            <option key={k} value={k}>
              {t(`ticket.kind.${k}`)}
            </option>
          ))}
        </select>
        <Badge variant="secondary">
          {ticket.assigneeId ? t("ops.ticketAssignee") : t("ops.ticketUnassigned")}
        </Badge>
      </div>

      <div className="space-y-3">
        {messages.map((m) => (
          <div
            key={m.id}
            className={cn(
              "rounded-lg border p-3",
              m.internal
                ? "border-amber-400 bg-amber-50 dark:bg-amber-950/20"
                : m.authorRole === "staff"
                  ? "bg-card"
                  : "bg-muted",
            )}
          >
            <div className="text-muted-foreground mb-1 flex items-center justify-between text-xs">
              <span>
                {m.authorRole === "staff"
                  ? m.authorName || t("support.staff")
                  : t("ops.ticketCustomer")}
                {m.internal && <span className="ml-2 font-medium">{t("ops.internalNote")}</span>}
              </span>
              <span>{formatDate(m.createdAt)}</span>
            </div>
            <p className="text-sm whitespace-pre-wrap">{m.body}</p>
          </div>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("ops.reply")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-2">
            <Textarea
              rows={4}
              required
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder={t("support.replyPlaceholder")}
            />
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={internal}
                onChange={(e) => setInternal(e.target.checked)}
              />
              {t("ops.internalNote")}
              <span className="text-muted-foreground text-xs">{t("ops.internalNoteHint")}</span>
            </label>
            <Button type="submit" disabled={reply.isPending}>
              {t("ops.reply")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
