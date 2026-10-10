import { forward } from "@/lib/proxy";

/** Adds a product to the signed-in shopper's wishlist. */
export async function POST(request: Request) {
  return forward(request, "/wishlist", "POST");
}
