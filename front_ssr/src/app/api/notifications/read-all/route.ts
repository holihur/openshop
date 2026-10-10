import { forward } from "@/lib/proxy";

/** Marks every notification as read. */
export async function POST(request: Request) {
  return forward(request, "/notifications/read-all", "POST");
}
