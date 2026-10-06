import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Minus, Plus, ShoppingBag, Tag, Trash2, X } from "lucide-react";
import { useState } from "react";

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
import { useShippingMethods } from "@lib/hooks/useShipping";
import { useAuth } from "@lib/auth";
import type { CouponPreview, Order, Payment } from "@lib/types";

export function CartPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const price = usePrice();
  const { currency } = useCurrency();
  const { data: cart, isLoading } = useCart();
  const updateItem = useUpdateCartItem();
  const removeItem = useRemoveCartItem();
  const clearCart = useClearCart();

  const [coupon, setCoupon] = useState("");
  const [applied, setApplied] = useState<CouponPreview | null>(null);
  const [addressId, setAddressId] = useState("");
  const [shippingMethodId, setShippingMethodId] = useState("");
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
  const chosenAddress =
    addressId || addresses?.find((a) => a.default)?.id || addresses?.[0]?.id || "";
  const chosenMethod =
    shippingMethods?.find((m) => m.id === shippingMethodId) ?? shippingMethods?.[0];
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
      toast.success(`Coupon ${result.code} applied`);
    },
    onError: (error: Error) => {
      setApplied(null);
      toast.error(error.message);
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
          provider: "mock",
          returnUrl: `${window.location.origin}/payment/result`,
        });
        return { order, payment };
      }
      // Creating the order reserves stock atomically on the server.
      const order = await api.post<Order>("/orders", {
        ...(couponCode ? { couponCode } : {}),
        ...(addressId ? { addressId } : {}),
        ...(shippingMethodId ? { shippingMethodId } : {}),
        currency,
      });
      // Then open a payment session with the (sandbox) provider.
      const payment = await api.post<Payment>("/payments", {
        orderId: order.id,
        provider: "mock",
        returnUrl: `${window.location.origin}/payment/result`,
      });
      return { order, payment };
    },
    onSuccess: ({ order, payment }) => {
      void queryClient.invalidateQueries({ queryKey: ["cart"] });
      void queryClient.invalidateQueries({ queryKey: ["orders"] });
      if (payment.redirectUrl) {
        window.location.href = payment.redirectUrl;
      } else if (!user && order.accessToken) {
        window.location.href = `/guest/orders/${order.accessToken}`;
      } else {
        navigate(`/orders/${order.id}`);
      }
    },
    onError: (error: Error) => toast.error(error.message),
  });

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  if (!cart || cart.items.length === 0) {
    return (
      <div className="py-20 text-center">
        <ShoppingBag className="text-muted-foreground mx-auto size-10" />
        <p className="mt-4 text-lg font-medium">Your cart is empty</p>
        <p className="text-muted-foreground">Add some products to get started.</p>
        <Button className="mt-6" asChild>
          <Link to="/products">Browse products</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Your cart</h1>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => clearCart.mutate()}
          disabled={clearCart.isPending}
        >
          Clear cart
        </Button>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
        <div className="space-y-3">
          {cart.items.map((item) => (
            <Card key={item.productId} className="flex-row items-center gap-4 p-4">
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
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-8"
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
            <CardTitle>Order summary</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Items</span>
              <span>{cart.totalCount}</span>
            </div>

            {!user && (
              <div className="space-y-2">
                <span className="text-sm font-medium">Delivery details</span>
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="Email for receipt"
                  required
                />
                <Input
                  value={guestAddress.recipient}
                  onChange={(e) => setGuestAddress({ ...guestAddress, recipient: e.target.value })}
                  placeholder="Recipient name"
                  required
                />
                <Input
                  value={guestAddress.phone}
                  onChange={(e) => setGuestAddress({ ...guestAddress, phone: e.target.value })}
                  placeholder="Phone"
                />
                <Input
                  value={guestAddress.line1}
                  onChange={(e) => setGuestAddress({ ...guestAddress, line1: e.target.value })}
                  placeholder="Address line"
                  required
                />
                <div className="grid grid-cols-2 gap-2">
                  <Input
                    value={guestAddress.city}
                    onChange={(e) => setGuestAddress({ ...guestAddress, city: e.target.value })}
                    placeholder="City"
                  />
                  <Input
                    value={guestAddress.postalCode}
                    onChange={(e) => setGuestAddress({ ...guestAddress, postalCode: e.target.value })}
                    placeholder="Postal code"
                  />
                </div>
              </div>
            )}

            {user && (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Shipping address</span>
                <Link to="/account/addresses" className="text-muted-foreground text-xs underline">
                  Manage
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
                  No address yet.{" "}
                  <Link to="/account/addresses" className="underline">
                    Add one
                  </Link>
                </p>
              )}
            </div>
            )}

            <div className="space-y-2">
              <span className="text-sm font-medium">Shipping</span>
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
                        {m.name}
                      </span>
                      <span className="text-muted-foreground">
                        {m.freeThresholdCents > 0 && cart.totalCents >= m.freeThresholdCents
                          ? "Free"
                          : price(m.flatRateCents)}
                      </span>
                    </label>
                  ))}
                </div>
              ) : (
                <p className="text-muted-foreground text-xs">No shipping options.</p>
              )}
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
                    aria-label="Remove coupon"
                  >
                    <X className="size-4" />
                  </button>
                </div>
              ) : (
                <div className="flex gap-2">
                  <Input
                    value={coupon}
                    onChange={(e) => setCoupon(e.target.value)}
                    placeholder="Coupon code"
                  />
                  <Button
                    variant="outline"
                    disabled={!coupon || previewCoupon.isPending}
                    onClick={() => previewCoupon.mutate(coupon)}
                  >
                    Apply
                  </Button>
                </div>
              )}
            </div>

            <Separator />
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Subtotal</span>
              <span>{price(cart.totalCents)}</span>
            </div>
            {applied && (
              <div className="flex justify-between text-sm text-emerald-600">
                <span>Discount</span>
                <span>-{price(applied.discountCents)}</span>
              </div>
            )}
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Shipping</span>
              <span>{price(shippingCents)}</span>
            </div>
            <p className="text-muted-foreground text-xs">Taxes are calculated at checkout.</p>
            <div className="flex justify-between text-base font-semibold">
              <span>Total</span>
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
              {checkout.isPending ? "Processing…" : "Checkout"}
            </Button>
            <p className="text-muted-foreground text-center text-xs">
              Stock is reserved atomically when you check out.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
