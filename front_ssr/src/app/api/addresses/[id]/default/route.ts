import { forward } from "@/lib/proxy";

/** Makes an address the default one used at checkout. */
export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return forward(request, `/addresses/${encodeURIComponent(id)}/default`, "POST");
}
