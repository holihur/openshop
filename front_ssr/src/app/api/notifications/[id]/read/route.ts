import { forward } from "@/lib/proxy";

export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return forward(request, `/notifications/${encodeURIComponent(id)}/read`, "POST");
}
