import { useEffect } from "react";

interface SeoOptions {
  title?: string;
  description?: string;
  image?: string;
  type?: string;
  /** Structured data (e.g. schema.org Product) injected as JSON-LD. */
  jsonLd?: Record<string, unknown>;
}

const SITE_NAME = "OpenShop";

function upsertMeta(attr: "name" | "property", key: string, content: string) {
  let el = document.head.querySelector<HTMLMetaElement>(`meta[${attr}="${key}"]`);
  if (!el) {
    el = document.createElement("meta");
    el.setAttribute(attr, key);
    document.head.appendChild(el);
  }
  el.setAttribute("content", content);
}

// useSeo sets per-page title, description, Open Graph tags and optional JSON-LD
// structured data. It restores the previous title when the page unmounts.
export function useSeo({ title, description, image, type = "website", jsonLd }: SeoOptions) {
  const json = jsonLd ? JSON.stringify(jsonLd) : "";
  useEffect(() => {
    const previousTitle = document.title;
    const fullTitle = title ? `${title} · ${SITE_NAME}` : SITE_NAME;
    document.title = fullTitle;
    upsertMeta("property", "og:title", fullTitle);
    upsertMeta("property", "og:site_name", SITE_NAME);
    upsertMeta("property", "og:type", type);
    upsertMeta("property", "og:url", window.location.href);
    if (description) {
      upsertMeta("name", "description", description);
      upsertMeta("property", "og:description", description);
    }
    if (image) {
      upsertMeta("property", "og:image", image);
    }

    let script: HTMLScriptElement | null = null;
    if (json) {
      script = document.createElement("script");
      script.type = "application/ld+json";
      script.text = json;
      document.head.appendChild(script);
    }

    return () => {
      document.title = previousTitle;
      script?.remove();
    };
  }, [title, description, image, type, json]);
}
