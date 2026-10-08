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
import { useI18n } from "@lib/i18n";
import { useAdminCustomers } from "@lib/hooks/useAdmin";
import { formatDate } from "@lib/format";
import { OpsModuleStats } from "@/components/ops-stats";

/** Customer directory: search, filter by status and open a customer. */
export function CustomersPage() {
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState("");
  const { data, isLoading } = useAdminCustomers(page, 20, { keyword, status });

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
      <OpsModuleStats module="customers" />
        <Input
          value={keyword}
          onChange={(e) => {
            setKeyword(e.target.value);
            setPage(1);
          }}
          placeholder={t("ops.searchCustomers")}
          className="max-w-xs"
        />
        <select
          value={status}
          onChange={(e) => {
            setStatus(e.target.value);
            setPage(1);
          }}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          <option value="">{t("ops.allStatuses")}</option>
          <option value="active">{t("common.active")}</option>
          <option value="disabled">{t("common.inactive")}</option>
        </select>
      </div>

      <Card className="py-0">
        <CardContent className="px-0">
          {isLoading ? (
            <div className="p-4">
              <Skeleton className="h-10 w-full" />
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("ops.customer")}</TableHead>
                  <TableHead>{t("ops.phone")}</TableHead>
                  <TableHead>{t("common.status")}</TableHead>
                  <TableHead>{t("ops.registeredAt")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.items.map((c) => (
                  <TableRow key={c.id}>
                    <TableCell>
                      <Link to={`/customers/${c.id}`} className="font-medium hover:underline">
                        {c.name || c.email}
                      </Link>
                      {c.name ? (
                        <div className="text-muted-foreground text-xs">{c.email}</div>
                      ) : null}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">{c.phone || "—"}</TableCell>
                    <TableCell>
                      <Badge variant={c.status === "active" ? "success" : "secondary"}>
                        {c.status === "active" ? t("common.active") : t("common.inactive")}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDate(c.createdAt)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {data && (
        <Pagination page={data.page} pageSize={data.pageSize} total={data.total} onChange={setPage} />
      )}
    </div>
  );
}
