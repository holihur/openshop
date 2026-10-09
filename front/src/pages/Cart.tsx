import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Minus, Plus, ShoppingBag, Tag, Trash2, X } from "lucide-react";
import { useState } from "react";
import { errorMessage } from "@lib/errors";
import { Button } from "@lib/components/ui/button";
import { Input } from "@lib/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Separator } from "@lib/components/ui/separator";
import { Skeleton } from "@lib/components/ui/skeleton";
import { api } from "@lib/api";
import { usePrice } from "@lib/hooks/usePrice";
import { useCurrency } from "@lib/currency";
import { useCart, useClearCart, useRemoveCartItem, useUpdateCartItem } from "@lib/hooks/useCart";
import { useAddresses } from "@lib/hooks/useAddresses";
import { useDeliveryEstimates, useShippingMethods } from "@lib/hooks/useShipping";
import { DeliveryEstimateLine, FreeShippingProgress } from "@lib/components/delivery-estimate";
import { PaymentQR } from "@lib/components/payment-qr";
import { usePaymentMethods } from "@lib/hooks/usePayment";
import { usePoints, useWallet } from "@lib/hooks/useLoyalty";
import { formatMoney } from "@lib/format";
import type { MessageKey } from "@lib/i18n/messages";
import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";
import type { CouponPreview, Order, Payment } from "@lib/types";

export function CartPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const { t } = useI18n();
  const price = usePrice();
  const { currency } = useCurrency();
  const { data: cart, isLoading } = useCart();
  const updateItem = useUpdateCartItem();
  const removeItem = useRemoveCartItem();
  const clearCart = useClearCart();

  const [coupon, setCoupon] = useState("");
  // Set when the provider returns a code to scan instead of a URL.
  const [qrPayment, setQrPayment] = useState<Payment | null>(null);
  const [applied, setApplied] = useState<CouponPreview | null>(null);
  const [addressId, setAddressId] = useState("");
  const [shippingMethodId, setShippingMethodId] = useState("");
  const [provider, setProvider] = useState("");
  const [useWalletBalance, setUseWalletBalance] = useState(false);
  const [pointsInput, setPointsInput] = useState("");
  const [email, setEmail] = useState("");
  const [guestAddress, setGuestAddress] = useState({
    recipient: "",
    phone: "",
    province: "",
    city: "",
    line1: "",
    postalCode: "",
  });
  const { data: addresses } = useAddresses();
  const { data: shippingMethods } = useShippingMethods();
  const { data: paymentMethods } = usePaymentMethods();
  const chosenProvider = provider || paymentMethods?.[0] || "mock";
  const { data: wallet } = useWallet(Boolean(user));
  const { data: pointsAccount } = usePoints(Boolean(user));
  const pointsValue = Math.max(0, Math.floor(Number(pointsInput) || 0));
  const chosenAddress =
    addressId || addresses?.find((a) => a.default)?.id || addresses?.[0]?.id || "";
  const chosenMethod =
    shippingMethods?.find((m) => m.id === shippingMethodId) ?? shippingMethods?.[0];
  // Server-side delivery expectations: shipping cost and arrival window per
  // method for this cart's subtotal and the chosen destination.
  const { data: deliveryEstimates } = useDeliveryEstimates(
    cart?.totalCents ?? 0,
    0,
    guestAddress.province || addresses?.find((a) => a.id === chosenAddress)?.province || "",
  );
  const chosenEstimate =
    deliveryEstimates?.find((e) => e.methodId === chosenMethod?.id) ?? deliveryEstimates?.[0];
  const estimateFor = (methodId: string) => deliveryEstimates?.find((e) => e.methodId === methodId);

  const shippingCents = chosenMethod
    ? chosenMethod.freeThresholdCents > 0 && cart && cart.totalCents >= chosenMethod.freeThresholdCents
      ? 0
      : chosenMethod.flatRateCents
    : 0;

  const previewCoupon = useMutation({
    mutationFn: (code: string) =>
      api.post<CouponPreview>("/coupons/preview", {
        code,
        subtotalCents: cart?.totalCents ?? 0,
      }),
    onSuccess: (result) => {
      setApplied(result);
      toast.success(t("cart.couponApplied", { code: result.code }));
    },
    onError: (error: Error) => {
      setApplied(null);
      toast.error(errorMessage(error));
    },
  });

  const checkout = useMutation({
    mutationFn: async ({ couponCode, addressId, shippingMethodId, email }: { couponCode: string; addressId: string; shippingMethodId: string; email: string }) => {
      if (!user) {
        // Guest checkout: the cart is keyed by the X-Guest-Id header.
        const order = await api.post<Order>("/orders", {
          ...(couponCode ? { couponCode } : {}),
          ...(shippingMethodId ? { shippingMethodId } : {}),
          email,
          currency,
          address: guestAddress,
        });
        if (order.accessToken) localStorage.setItem("openshop.guestOrderToken", order.accessToken);
        const payment = await api.post<Payment>(`/guest/orders/${order.accessToken}/pay`, {
          provider: chosenProvider,
          returnUrl: `${window.location.origin}/payment/result`,
        });
        return { order, payment };
      }
      // Creating the order reserves stock atomically on the server.
      const order = await api.post<Order>("/orders", {
        ...(couponCode ? { couponCode } : {}),
        ...(addressId ? { addressId } : {}),
        ...(shippingMethodId ? { shippingMethodId } : {}),
        ...(useWalletBalance ? { useWallet: true } : {}),
        ...(pointsValue > 0 ? { points: pointsValue } : {}),
        currency,
      });
      // Then open a payment session with the (sandbox) provider.
      const payment = await api.post<Payment>("/payments", {
        orderId: order.id,
        provider: chosenProvider,
        returnUrl: `${window.location.origin}/payment/result`,
      });
      return { order, payment };
    },
    onSuccess: ({ order, payment }) => {
      void queryClient.invalidateQueries({ queryKey: ["cart"] });
      void queryClient.invalidateQueries({ queryKey: ["orders"] });
      // A provider that returns a code to scan (WeChat Native) has no URL to
      // open; the QR is shown instead and the webhook confirms the payment.
      if (payment.qrSvg) {
        setQrPayment(payment);
        return;
      }
      if (payment.redirectUrl) {
        window.location.href = payment.redirectUrl;
      } else if (!user && order.accessToken) {
        window.location.href = `/guest/orders/${order.accessToken}`;
      } else {
        navigate(`/orders/${order.id}`);
      }
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-40" />
        <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 rounded-xl border p-4">
                <Skeleton className="size-20 shrink-0 rounded-md" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-4 w-2/3" />
                  <Skeleton className="h-3 w-1/3" />
                </div>
                <Skeleton className="h-8 w-24" />
              </div>
            ))}
          </div>
          <Skeleton className="h-72 w-full" />
        </div>
      </div>
    );
  }

  if (!cart || cart.items.length === 0) {
    return (
      <div className="py-20 text-center">
        <ShoppingBag className="text-muted-foreground mx-auto size-10" />
        <h1 className="mt-4 text-lg font-medium">{t("cart.emptyTitle")}</h1>
        <p className="text-muted-foreground">{t("cart.emptyHint")}</p>
        <Button className="mt-6" asChild>
          <Link to="/products">{t("cart.browse")}</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t("cart.yourCart")}</h1>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => clearCart.mutate()}
          disabled={clearCart.isPending}
        >
          {t("cart.clear")}
        </Button>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
        <div className="space-y-3">
          {cart.items.map((item) => (
            <Card key={item.productId} className="flex-row flex-wrap items-center gap-4 p-4">
              <Link to={`/products/${item.productId}`} className="shrink-0">
                <div className="bg-muted size-20 overflow-hidden rounded-md">
                  {item.coverImage ? (
                    <img src={item.coverImage} alt={item.title} className="size-full object-cover" />
                  ) : null}
                </div>
              </Link>
              <div className="min-w-0 flex-1">
                <Link to={`/products/${item.productId}`} className="font-medium hover:underline">
                  {item.title}
                </Link>
                {item.variantName && (
                  <p className="text-muted-foreground text-xs">{item.variantName}</p>
                )}
                <p className="text-muted-foreground text-sm">
                  {price(item.priceCents)}
                </p>
              </div>
              <div className="flex items-center rounded-md border">
                {qrPayment ? <PaymentQR svg={qrPayment.qrSvg} provider={qrPayment.provider} /> : null}
            <Button
                  variant="ghost"
                  size="icon"
                  className="size-8"
                  aria-label={t("cart.decrease")}
                  disabled={item.quantity <= 1 || updateItem.isPending}
                  onClick={() =>
                    updateItem.mutate({ productId: item.productId, variantId: item.variantId, quantity: item.quantity - 1 })
                  }
                >
                  <Minus className="size-3" />
                </Button>
                <span className="w-8 text-center text-sm">{item.quantity}</span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-8"
                  aria-label={t("cart.increase")}
                  disabled={updateItem.isPending}
                  onClick={() =>
                    updateItem.mutate({ productId: item.productId, variantId: item.variantId, quantity: item.quantity + 1 })
                  }
                >
                  <Plus className="size-3" />
                </Button>
              </div>
              <Button
                variant="ghost"
                size="icon"
                aria-label={t("cart.remove")}
                disabled={removeItem.isPending}
                onClick={() => removeItem.mutate({ productId: item.productId, variantId: item.variantId })}
              >
                <Trash2 className="size-4" />
              </Button>
            </Card>
          ))}
        </div>

        <Card className="h-fit">
          <CardHeader>
            <CardTitle>{t("cart.orderSummary")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t("cart.items")}</span>
              <span>{cart.totalCount}</span>
            </div>

            {!user && (
              <div className="space-y-2">
                <span className="text-sm font-medium">{t("cart.deliveryDetails")}</span>
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder={t("cart.emailForReceipt")}
                  required
                />
                <Input
                  value={guestAddress.recipient}
                  onChange={(e) => setGuestAddress({ ...guestAddress, recipient: e.target.value })}
                  placeholder={t("cart.recipientName")}
                  required
                />
                <Input
                  value={guestAddress.phone}
                  onChange={(e) => setGuestAddress({ ...guestAddress, phone: e.target.value })}
                  placeholder={t("cart.phone")}
                />
                <Input
                  value={guestAddress.line1}
                  onChange={(e) => setGuestAddress({ ...guestAddress, line1: e.target.value })}
                  placeholder={t("cart.addressLine")}
                  required
                />
                <div className="grid grid-cols-2 gap-2">
                  <Input
                    value={guestAddress.city}
                    onChange={(e) => setGuestAddress({ ...guestAddress, city: e.target.value })}
                    placeholder={t("cart.city")}
                  />
                  <Input
                    value={guestAddress.postalCode}
                    onChange={(e) => setGuestAddress({ ...guestAddress, postalCode: e.target.value })}
                    placeholder={t("cart.postalCode")}
                  />
                </div>
              </div>
            )}

            {user && (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">{t("orders.shippingAddress")}</span>
                <Link to="/account/addresses" className="text-muted-foreground text-xs underline">
                  {t("cart.manage")}
                </Link>
              </div>
              {addresses && addresses.length > 0 ? (
                <div className="space-y-1">
                  {addresses.map((a) => (
                    <label
                      key={a.id}
                      className="flex cursor-pointer items-start gap-2 rounded-md border p-2 text-sm"
                    >
                      <input
                        type="radio"
                        name="shipping-address"
                        className="mt-1"
                        checked={chosenAddress === a.id}
                        onChange={() => setAddressId(a.id)}
                      />
                      <span>
                        <span className="font-medium">{a.recipient}</span>
                        <span className="text-muted-foreground">
                          {" "}
                          {[a.province, a.city, a.district, a.line1].filter(Boolean).join(" ")}
                        </span>
                      </span>
                    </label>
                  ))}
                </div>
              ) : (
                <p className="text-muted-foreground text-xs">
                  {t("cart.noAddress")}{" "}
                  <Link to="/account/addresses" className="underline">
                    {t("cart.addOne")}
                  </Link>
                </p>
              )}
            </div>
            )}

            <div className="space-y-2">
              <span className="text-sm font-medium">{t("cart.shipping")}</span>
              {shippingMethods && shippingMethods.length > 0 ? (
                <div className="space-y-1">
                  {shippingMethods.map((m) => (
                    <label
                      key={m.id}
                      className="flex cursor-pointer items-center justify-between gap-2 rounded-md border p-2 text-sm"
                    >
                      <span className="flex items-center gap-2">
                        <input
                          type="radio"
                          name="shipping-method"
                          checked={chosenMethod?.id === m.id}
                          onChange={() => setShippingMethodId(m.id)}
                        />
                        <span>
                          {m.name}
                          <DeliveryEstimateLine
                            estimate={estimateFor(m.id)}
                            className="text-muted-foreground block text-xs"
                          />
                        </span>
                      </span>
                      <span className="text-muted-foreground">
                        {m.freeThresholdCents > 0 && cart.totalCents >= m.freeThresholdCents
                          ? t("cart.free")
                          : price(m.flatRateCents)}
                      </span>
                    </label>
                  ))}
                </div>
              ) : (
                <p className="text-muted-foreground text-xs">{t("cart.noShipping")}</p>
              )}
            </div>

            <div className="space-y-2">
              <span className="text-sm font-medium">{t("cart.paymentMethod")}</span>
              <div className="space-y-1">
                {(paymentMethods ?? ["mock"]).map((name) => (
                  <label
                    key={name}
                    className="flex cursor-pointer items-center gap-2 rounded-md border p-2 text-sm"
                  >
                    <input
                      type="radio"
                      name="payment-method"
                      checked={chosenProvider === name}
                      onChange={() => setProvider(name)}
                    />
                    {t(`payment.method.${name}` as MessageKey)}
                  </label>
                ))}
              </div>
            </div>

            <div className="space-y-2">
              {applied ? (
                <div className="bg-muted flex items-center justify-between rounded-md px-3 py-2 text-sm">
                  <span className="flex items-center gap-2">
                    <Tag className="size-4" />
                    {applied.code}
                  </span>
                  <button
                    onClick={() => {
                      setApplied(null);
                      setCoupon("");
                    }}
                    aria-label={t("cart.removeCoupon")}
                  >
                    <X className="size-4" />
                  </button>
                </div>
              ) : (
                <div className="flex gap-2">
                  <Input
                    value={coupon}
                    onChange={(e) => setCoupon(e.target.value)}
                    placeholder={t("cart.couponCode")}
                  />
                  <Button
                    variant="outline"
                    disabled={!coupon || previewCoupon.isPending}
                    onClick={() => previewCoupon.mutate(coupon)}
                  >
                    {t("cart.applyCoupon")}
                  </Button>
                </div>
              )}
            </div>

            <Separator />
            {chosenEstimate && (
              <FreeShippingProgress
                subtotalCents={cart.totalCents}
                thresholdCents={chosenEstimate.freeThresholdCents}
                remainingCents={chosenEstimate.freeRemainingCents}
                format={price}
              />
            )}
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t("cart.subtotal")}</span>
              <span>{price(cart.totalCents)}</span>
            </div>
            {applied && (
              <div className="flex justify-between text-sm text-emerald-600">
                <span>{t("cart.discount")}</span>
                <span>-{price(applied.discountCents)}</span>
              </div>
            )}
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t("cart.shipping")}</span>
              <span>{price(chosenEstimate?.priceCents ?? shippingCents)}</span>
            </div>
            <p className="text-muted-foreground text-xs">{t("cart.taxesAtCheckout")}</p>
            {user && wallet && wallet.balanceCents > 0 && (
              <label className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={useWalletBalance}
                    onChange={(e) => setUseWalletBalance(e.target.checked)}
                  />
                  {t("cart.useWallet")}
                </span>
                <span className="text-muted-foreground">
                  {formatMoney(wallet.balanceCents, wallet.currency)}
                </span>
              </label>
            )}
            {user && pointsAccount && pointsAccount.balance > 0 && (
              <div className="flex items-center justify-between text-sm">
                <span>{t("cart.usePoints", { balance: pointsAccount.balance })}</span>
                <input
                  type="number"
                  min="0"
                  max={pointsAccount.balance}
                  value={pointsInput}
                  onChange={(e) => setPointsInput(e.target.value)}
                  placeholder="0"
                  className="border-input bg-background h-8 w-24 rounded-md border px-2 text-sm"
                />
              </div>
            )}
            <div className="flex justify-between text-base font-semibold">
              <span>{t("cart.total")}</span>
              <span>{price((applied?.totalCents ?? cart.totalCents) + shippingCents)}</span>
            </div>
            <Button
              className="w-full"
              size="lg"
              disabled={
                checkout.isPending ||
                (!user && (email.trim() === "" || guestAddress.recipient.trim() === "" || guestAddress.line1.trim() === ""))
              }
              onClick={() =>
                checkout.mutate({
                  couponCode: applied?.code ?? "",
                  addressId: chosenAddress,
                  shippingMethodId: chosenMethod?.id ?? "",
                  email,
                })
              }
            >
              {checkout.isPending ? t("cart.processing") : t("cart.checkout")}
            </Button>
            <p className="text-muted-foreground text-center text-xs">{t("cart.stockNote")}</p>
            <div className="text-center">
              <Button variant="link" size="sm" asChild>
                <Link to="/products">{t("cart.continueShopping")}</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
