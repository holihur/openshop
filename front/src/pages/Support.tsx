import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { LifeBuoy } from "lucide-react";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Skeleton } from "@lib/components/ui/skeleton";
import { Textarea } from "@lib/components/ui/textarea";
import { useAuth } from "@lib/auth";
import { useCreateTicket, useMyTickets } from "@lib/hooks/useTickets";
import { useI18n } from "@lib/i18n";
import { formatDate } from "@lib/format";
import type { Ticket, TicketStatus } from "@lib/types";

const STATUS_VARIANT: Record<TicketStatus, "default" | "secondary" | "success" | "warning"> = {
  open: "warning",
  pending: "default",
  resolved: "success",
  closed: "secondary",
};

export function TicketStatusBadge({ status }: { status: TicketStatus }) {
  const { t } = useI18n();
  return <Badge variant={STATUS_VARIANT[status] ?? "secondary"}>{t(`ticket.status.${status}`)}</Badge>;
}

function NewTicketForm({ onCreated }: { onCreated?: (ticket: Ticket) => void }) {
  const { t } = useI18n();
  const { user } = useAuth();
  const create = useCreateTicket();
  const [subject, setSubject] = useState("");
  const [kind, setKind] = useState("other");
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [body, setBody] = useState("");
  const [reference, setReference] = useState("");

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    create.mutate(
      { subject, kind, body, email: user ? undefined : email, name: user ? undefined : name },
      {
        onSuccess: (ticket) => {
          setSubject("");
          setBody("");
          setReference(ticket.reference);
          onCreated?.(ticket);
        },
      },
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t("support.newTicket")}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="space-y-4">
          {!user && (
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="ticket-name">{t("support.name")}</Label>
                <Input id="ticket-name" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="ticket-email">{t("support.email")}</Label>
                <Input
                  id="ticket-email"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
            </div>
          )}
          <div className="grid gap-4 sm:grid-cols-[1fr_200px]">
            <div className="space-y-2">
              <Label htmlFor="ticket-subject">{t("support.subject")}</Label>
              <Input
                id="ticket-subject"
                required
                maxLength={300}
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="ticket-kind">{t("support.kind")}</Label>
              <select
                id="ticket-kind"
                value={kind}
                onChange={(e) => setKind(e.target.value)}
                className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
              >
                <option value="presale">{t("ticket.kind.presale")}</option>
                <option value="postsale">{t("ticket.kind.postsale")}</option>
                <option value="other">{t("ticket.kind.other")}</option>
              </select>
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="ticket-body">{t("support.message")}</Label>
            <Textarea
              id="ticket-body"
              required
              rows={5}
              value={body}
              onChange={(e) => setBody(e.target.value)}
            />
          </div>
          {reference && (
            <p className="text-sm text-emerald-600">
              {t("support.created", { reference })}
            </p>
          )}
          <Button type="submit" disabled={create.isPending}>
            {create.isPending ? t("common.saving") : t("support.submit")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}

export function SupportPage() {
  const { t } = useI18n();
  const { user } = useAuth();
  const { data, isLoading } = useMyTickets();
  const tickets = data?.items ?? [];

  return (
    <div className="mx-auto max-w-3xl space-y-6 py-8">
      <div className="flex items-center gap-2">
        <LifeBuoy className="size-6" />
        <h1 className="text-2xl font-semibold">{t("support.title")}</h1>
      </div>
      <p className="text-muted-foreground text-sm">{t("support.subtitle")}</p>

      <NewTicketForm />

      {user && (
        <div className="space-y-3">
          <h2 className="text-lg font-medium">{t("support.myTickets")}</h2>
          {isLoading ? (
            <Skeleton className="h-24 w-full" />
          ) : tickets.length === 0 ? (
            <p className="text-muted-foreground text-sm">{t("support.empty")}</p>
          ) : (
            <ul className="space-y-2">
              {tickets.map((ticket) => (
                <li key={ticket.id}>
                  <Link
                    to={`/support/${ticket.id}`}
                    className="hover:bg-accent flex items-center justify-between gap-3 rounded-md border p-3"
                  >
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-muted-foreground text-xs">{ticket.reference}</span>
                        <TicketStatusBadge status={ticket.status} />
                      </div>
                      <p className="truncate font-medium">{ticket.subject}</p>
                    </div>
                    <span className="text-muted-foreground shrink-0 text-xs">
                      {formatDate(ticket.updatedAt)}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
