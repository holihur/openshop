import { Skeleton } from "@lib/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@lib/components/ui/table";
import { useI18n } from "@lib/i18n";
import { useCouponRedemptions } from "@lib/hooks/useAdmin";
import { formatDate, formatMoney } from "@lib/format";

/** A coupon's usage history: who redeemed it, on which order and how much. */
export function CouponUsage({ couponId }: { couponId: string }) {
  const { t } = useI18n();
  const { data, isLoading } = useCouponRedemptions(couponId);

  if (isLoading) {
    return <Skeleton className="h-20 w-full" />;
  }
  if (!data || data.items.length === 0) {
    return <p className="text-muted-foreground text-sm">{t("ops.noUsage")}</p>;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("orders.orderNo")}</TableHead>
          <TableHead>{t("ops.customer")}</TableHead>
          <TableHead>{t("ops.discount")}</TableHead>
          <TableHead>{t("ops.date")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {data.items.map((r) => (
          <TableRow key={r.id}>
            <TableCell className="font-mono text-xs">
              {r.orderNo || r.orderId.slice(0, 8)}
            </TableCell>
            <TableCell>{r.userEmail || r.userId.slice(0, 8)}</TableCell>
            <TableCell>{formatMoney(r.discountCents)}</TableCell>
            <TableCell className="text-muted-foreground text-sm">
              {formatDate(r.createdAt)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
