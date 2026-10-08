import { useState } from "react";
import { Link } from "react-router-dom";

import { Badge } from "@lib/components/ui/badge";
import { Card, CardContent } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Skeleton } from "@lib/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@lib/components/ui/table";
import { Pagination } from "@lib/components/pagination";
import { useAdminTickets } from "@lib/hooks/useTickets";
import { useI18n } from "@lib/i18n";
import { formatDate } from "@lib/format";
import { OpsModuleStats } from "@/components/ops-stats";
import type { TicketPriority, TicketStatus } from "@lib/types";

const STATUS_VARIANT: Record<TicketStatus, "default" | "secondary" | "success" | "warning"> = {
  open: "warning",
  pending: "default",
  resolved: "success",
  closed: "secondary",
};

const PRIORITY_VARIANT: Record<TicketPriority, "secondary" | "default" | "warning" | "destructive"> =
  {
    low: "secondary",
    normal: "default",
    high: "warning",
    urgent: "destructive",
  };

export function TicketsPage() {
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const [kind, setKind] = useState("");
  const [assignee, setAssignee] = useState("");
  const [q, setQ] = useState("");
  const { data, isLoading } = useAdminTickets({ page, pageSize: 20, status, kind, assignee, q });

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t("ops.tickets")}</h1>
        <p className="text-muted-foreground text-sm">{t("ops.ticketsSubtitle")}</p>
      </div>

      <OpsModuleStats module="tickets" />

      <div className="flex flex-wrap gap-2">
        <Input
          value={q}
          onChange={(e) => {
            setQ(e.target.value);
            setPage(1);
          }}
          placeholder={t("common.search")}
          className="max-w-xs"
        />
        <select
          aria-label={t("ops.ticketStatus")}
          value={status}
          onChange={(e) => {
            setStatus(e.target.value);
            setPage(1);
          }}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          <option value="">{t("ops.allStatuses")}</option>
          {(["open", "pending", "resolved", "closed"] as const).map((s) => (
            <option key={s} value={s}>
              {t(`ticket.status.${s}`)}
            </option>
          ))}
        </select>
        <select
          aria-label={t("ops.ticketKind")}
          value={kind}
          onChange={(e) => {
            setKind(e.target.value);
            setPage(1);
          }}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          <option value="">{t("ops.ticketKind")}</option>
          {(["presale", "postsale", "other"] as const).map((k) => (
            <option key={k} value={k}>
              {t(`ticket.kind.${k}`)}
            </option>
          ))}
        </select>
        <select
          aria-label={t("ops.ticketAssignee")}
          value={assignee}
          onChange={(e) => {
            setAssignee(e.target.value);
            setPage(1);
          }}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          <option value="">{t("ops.ticketAssignee")}</option>
          <option value="me">{t("ops.assignToMe")}</option>
          <option value="unassigned">{t("ops.ticketUnassigned")}</option>
        </select>
      </div>

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <Card className="py-0">
          <CardContent className="px-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("support.subject")}</TableHead>
                  <TableHead>{t("ops.ticketCustomer")}</TableHead>
                  <TableHead>{t("ops.ticketKind")}</TableHead>
                  <TableHead>{t("ops.ticketPriority")}</TableHead>
                  <TableHead>{t("ops.ticketStatus")}</TableHead>
                  <TableHead>{t("ops.date")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.items.map((ticket) => (
                  <TableRow key={ticket.id}>
                    <TableCell>
                      <Link to={`/tickets/${ticket.id}`} className="font-medium hover:underline">
                        {ticket.subject}
                      </Link>
                      <div className="text-muted-foreground text-xs">{ticket.reference}</div>
                    </TableCell>
                    <TableCell className="text-sm">
                      {ticket.name || ticket.email}
                      <div className="text-muted-foreground text-xs">{ticket.email}</div>
                    </TableCell>
                    <TableCell className="text-sm">{t(`ticket.kind.${ticket.kind}`)}</TableCell>
                    <TableCell>
                      <Badge variant={PRIORITY_VARIANT[ticket.priority] ?? "secondary"}>
                        {t(`ticket.priority.${ticket.priority}`)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={STATUS_VARIANT[ticket.status] ?? "secondary"}>
                        {t(`ticket.status.${ticket.status}`)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDate(ticket.updatedAt)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <Pagination
        page={data?.page ?? page}
        pageSize={data?.pageSize ?? 20}
        total={data?.total ?? 0}
        onChange={setPage}
      />
    </div>
  );
}
