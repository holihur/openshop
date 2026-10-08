import { describe, expect, it } from "vitest";

import { responsiveSrcSet } from "@lib/media";

describe("responsiveSrcSet", () => {
  it("builds a srcset for backend-served uploads", () => {
    const out = responsiveSrcSet("http://localhost:8080/uploads/a.jpg");
    expect(out).toBe(
      "http://localhost:8080/uploads/a.jpg?w=320 320w, " +
        "http://localhost:8080/uploads/a.jpg?w=640 640w, " +
        "http://localhost:8080/uploads/a.jpg?w=1280 1280w",
    );
  });

  it("appends to an existing query string", () => {
    expect(responsiveSrcSet("/uploads/a.png?v=2")).toContain("/uploads/a.png?v=2&w=320 320w");
  });

  // External/CDN images are not resized by the backend, so no srcset is offered
  // and the browser uses the plain src.
  it("returns undefined for external or missing urls", () => {
    expect(responsiveSrcSet("https://cdn.example.com/a.jpg")).toBeUndefined();
    expect(responsiveSrcSet("")).toBeUndefined();
    expect(responsiveSrcSet(null)).toBeUndefined();
    expect(responsiveSrcSet(undefined)).toBeUndefined();
  });
});
