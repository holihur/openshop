import { redirect } from "next/navigation";

import { apiGet } from "@/lib/api";
import { AccountNav } from "@/components/account-nav";
import { AddressBook } from "@/components/address-book";
import { translator } from "@/lib/i18n";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { Address } from "@/lib/types";

/** The address book used at checkout. */
export default async function AddressesPage() {
  const shopper = await getShopper();
  if (!shopper.token) redirect("/login");

  const locale = await resolveLocale();
  const t = translator(locale);
  const addresses = await apiGet<Address[]>("/addresses", { token: shopper.token }).catch(
    () => [] as Address[],
  );

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("account.addresses")}</h1>
      <AccountNav locale={locale} current="/account/addresses" />
      <AddressBook
        addresses={addresses}
        labels={{
          add: t("account.addAddress"),
          recipient: t("checkout.recipient"),
          phone: t("checkout.phone"),
          province: t("checkout.province"),
          city: t("checkout.city"),
          district: t("checkout.district"),
          line1: t("checkout.line1"),
          postalCode: t("checkout.postalCode"),
          save: t("common.save"),
          saving: t("cart.updating"),
          remove: t("cart.remove"),
          makeDefault: t("account.makeDefault"),
          isDefault: t("account.isDefault"),
          empty: t("account.noAddresses"),
          failed: t("error.generic"),
        }}
      />
    </div>
  );
}
