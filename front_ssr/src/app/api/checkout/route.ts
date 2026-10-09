import { NextResponse } from "next/server";
import { cookies } from "next/headers";

import { API_BASE, ACCESS_COOKIE, GUEST_COOKIE } from "@/lib/api";

interface OrderResponse {
  id: string;
  orderNo: string;
  accessToken?: string;
}

interface PaymentResponse {
  redirectUrl?: string;
  // Set when the provider returned something to scan (WeChat Native) rather
  // than a URL to open.
  qrSvg?: string;
}

/**
 * Places an order and starts the payment in one step, server side.
 *
 * Doing it here rather than in the browser means the happy path works with
 * JavaScript disabled until the very last click, and the API credentials never
 * leave the server.
 */
export async function POST(request: Request) {
  const jar = await cookies();
  const token = jar.get(ACCESS_COOKIE)?.value;
  const guestId = jar.get(GUEST_COOKIE)?.value;

  const body = (await request.json().catch(() => ({}))) as {
    addressId?: string;
    email?: string;
    shippingMethodId?: string;
    provider?: string;
    address?: Record<string, string>;
    origin?: string;
  };

  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (guestId) headers["X-Guest-Id"] = guestId;

  const orderBody: Record<string, unknown> = { currency: "CNY" };
  if (body.shippingMethodId) orderBody.shippingMethodId = body.shippingMethodId;
  if (body.addressId) orderBody.addressId = body.addressId;
  if (body.email) orderBody.email = body.email;
  if (body.address) orderBody.address = body.address;

  const orderRes = await fetch(`${API_BASE}/api/v1/orders`, {
    method: "POST",
    headers,
    body: JSON.stringify(orderBody),
    cache: "no-store",
  });
  const orderPayload = (await orderRes.json().catch(() => ({}))) as {
    data?: OrderResponse;
    error?: { message?: string };
  };
  if (!orderRes.ok || !orderPayload.data) {
    return NextResponse.json(
      { error: orderPayload.error ?? { code: "order_failed", message: "Could not place the order" } },
      { status: orderRes.status || 400 },
    );
  }
  const order = orderPayload.data;
  const returnUrl = `${body.origin ?? ""}/checkout/result`;

  // A guest pays through the order's own access token; a signed-in customer
  // goes through the authenticated payment endpoint.
  const payPath = order.accessToken
    ? `/guest/orders/${encodeURIComponent(order.accessToken)}/pay`
    : "/payments";
  const payBody = order.accessToken
    ? { provider: body.provider, returnUrl }
    : { orderId: order.id, provider: body.provider, returnUrl };

  const payRes = await fetch(`${API_BASE}/api/v1${payPath}`, {
    method: "POST",
    headers,
    body: JSON.stringify(payBody),
    cache: "no-store",
  });
  const payPayload = (await payRes.json().catch(() => ({}))) as {
    data?: PaymentResponse;
    error?: { message?: string };
  };
  if (!payRes.ok || !payPayload.data) {
    return NextResponse.json(
      {
        error: payPayload.error ?? {
          code: "payment_failed",
          message: "The order was created but the payment could not start",
        },
        data: { orderId: order.id, orderNo: order.orderNo },
      },
      { status: payRes.status || 400 },
    );
  }

  return NextResponse.json({
    data: {
      orderId: order.id,
      orderNo: order.orderNo,
      redirectUrl: payPayload.data.redirectUrl ?? `${returnUrl}?order_no=${order.orderNo}`,
      qrSvg: payPayload.data.qrSvg,
    },
  });
}
