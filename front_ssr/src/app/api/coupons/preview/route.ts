import { forward } from "@/lib/proxy";

/** Prices a coupon against the current cart before it is committed. */
export async function POST(request: Request) {
  return forward(request, "/coupons/preview", "POST");
}
