import { forward } from "@/lib/proxy";

/** Replies to a support ticket. */
export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return forward(request, `/tickets/${encodeURIComponent(id)}/messages`, "POST");
}
