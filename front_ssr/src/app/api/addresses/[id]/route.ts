import { forward } from "@/lib/proxy";

export async function PATCH(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return forward(request, `/addresses/${encodeURIComponent(id)}`, "PATCH");
}

export async function DELETE(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return forward(request, `/addresses/${encodeURIComponent(id)}`, "DELETE");
}
