import { useState } from "react";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent } from "@lib/components/ui/card";
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
import { useOutboxEvents, useOutboxStats, useReplayOutboxEvent } from "@lib/hooks/useOutbox";
import { useI18n } from "@lib/i18n";
import { formatDate } from "@lib/format";
import type { OutboxEvent } from "@lib/hooks/useOutbox";

const STATUS_VARIANT: Record<OutboxEvent["status"], "secondary" | "default" | "success" | "destructive"> =
  {
    pending: "default",
    processing: "secondary",
    published: "success",
    failed: "destructive",
  };

export function OutboxPage() {
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("failed");
  const { data: stats } = useOutboxStats();
  const { data, isLoading } = useOutboxEvents({ status, page });
  const replay = useReplayOutboxEvent();

  const cards = [
    { key: "ops.outboxPending", value: stats?.pending ?? 0 },
    { key: "ops.outboxProcessing", value: stats?.processing ?? 0 },
    { key: "ops.outboxFailed", value: stats?.failed ?? 0, alert: (stats?.failed ?? 0) > 0 },
    { key: "ops.outboxPublished", value: stats?.published ?? 0 },
  ] as const;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t("ops.outbox")}</h1>
        <p className="text-muted-foreground text-sm">{t("ops.outboxSubtitle")}</p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {cards.map((card) => (
          <Card key={card.key}>
            <CardContent className="pt-4">
              <p className="text-muted-foreground text-xs">{t(card.key)}</p>
              <p
                className={
                  "alert" in card && card.alert
                    ? "text-2xl font-semibold text-destructive"
                    : "text-2xl font-semibold"
                }
              >
                {card.value}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>

      {stats?.oldestPendingAgeSeconds !== undefined && stats.oldestPendingAgeSeconds > 0 && (
        <p className="text-muted-foreground text-xs">
          {t("ops.outboxOldest", { seconds: stats.oldestPendingAgeSeconds })}
        </p>
      )}

      <div className="flex flex-wrap gap-2">
        <select
          aria-label={t("common.status")}
          value={status}
          onChange={(e) => {
            setStatus(e.target.value);
            setPage(1);
          }}
          className="border-input bg-background h-9 rounded-md px-3 text-sm"
        >
          <option value="">{t("ops.allStatuses")}</option>
          {(["pending", "processing", "published", "failed"] as const).map((s) => (
            <option key={s} value={s}>
              {t(`ops.outboxStatus.${s}`)}
            </option>
          ))}
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
                  <TableHead>{t("ops.outboxSubject")}</TableHead>
                  <TableHead>{t("common.status")}</TableHead>
                  <TableHead>{t("ops.outboxAttempts")}</TableHead>
                  <TableHead>{t("ops.outboxLastError")}</TableHead>
                  <TableHead>{t("ops.outboxCreated")}</TableHead>
                  <TableHead className="text-right">{t("common.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.items.map((evt) => (
                  <TableRow key={evt.id}>
                    <TableCell className="font-mono text-xs">{evt.subject}</TableCell>
                    <TableCell>
                      <Badge variant={STATUS_VARIANT[evt.status] ?? "secondary"}>
                        {t(`ops.outboxStatus.${evt.status}`)}
                      </Badge>
                    </TableCell>
                    <TableCell>{evt.attempts}</TableCell>
                    <TableCell className="text-muted-foreground max-w-md truncate text-xs">
                      {evt.lastError || "—"}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDate(evt.createdAt)}
                    </TableCell>
                    <TableCell className="text-right">
                      {evt.status === "failed" && (
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={replay.isPending}
                          onClick={() => replay.mutate(evt.id)}
                        >
                          {t("ops.outboxReplay")}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
                {data && data.items.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-muted-foreground text-center text-sm">
                      {t("ops.outboxEmpty")}
                    </TableCell>
                  </TableRow>
                )}
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
