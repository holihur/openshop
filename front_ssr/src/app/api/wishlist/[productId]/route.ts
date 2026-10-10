import { forward } from "@/lib/proxy";

/** Removes a product from the wishlist. */
export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ productId: string }> },
) {
  const { productId } = await params;
  return forward(request, `/wishlist/${encodeURIComponent(productId)}`, "DELETE");
}
