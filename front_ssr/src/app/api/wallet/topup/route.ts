import { forward } from "@/lib/proxy";

/** Tops up the wallet through the configured payment gateway. */
export async function POST(request: Request) {
  return forward(request, "/wallet/topup", "POST");
}
