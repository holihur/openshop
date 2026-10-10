import { forward } from "@/lib/proxy";

/** Opens a support ticket. Signed-in shoppers are linked to it automatically. */
export async function POST(request: Request) {
  return forward(request, "/tickets", "POST");
}
