import { PaymentConfirmer } from "@/components/payment-confirmer";
import { translator } from "@/lib/i18n";
import { resolveLocale } from "@/app/layout";

/**
 * Where the payment provider sends the shopper back. The status is confirmed by
 * a client island because confirmation is a POST, not a page load.
 */
export default async function PaymentResultPage({
  searchParams,
}: {
  searchParams: Promise<{ payment_ref?: string; provider?: string; order_no?: string }>;
}) {
  const params = await searchParams;
  const locale = await resolveLocale();
  const t = translator(locale);

  return (
    <PaymentConfirmer
      providerRef={params.payment_ref ?? ""}
      provider={params.provider ?? "mock"}
      orderNo={params.order_no ?? ""}
      locale={locale}
      labels={{
        confirming: t("checkout.confirming"),
        paid: t("checkout.paid"),
        failed: t("checkout.failed"),
        failReason: t("error.generic"),
      }}
    />
  );
}
