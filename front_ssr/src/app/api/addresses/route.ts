import { forward } from "@/lib/proxy";

/** Creates a delivery address for the signed-in shopper. */
export async function POST(request: Request) {
  return forward(request, "/addresses", "POST");
}
