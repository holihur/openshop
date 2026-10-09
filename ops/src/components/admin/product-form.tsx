import { useState, type ChangeEvent, type FormEvent } from "react";
import { Upload, X } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Textarea } from "@lib/components/ui/textarea";
import { useCategories, useCreateProduct, useUpdateProduct, useUploadImage } from "@lib/hooks/useAdmin";
import { useI18n } from "@lib/i18n";
import { localeNames, supportedLocales } from "@lib/i18n/messages";
import type { Product } from "@lib/types";

interface FormState {
  title: string;
  // Per-locale titles; an empty value falls back to the base title.
  names: Record<string, string>;
  description: string;
  price: string; // in major units, converted to cents on submit
  stock: string;
  weight: string; // grams
  status: string;
  categoryId: string;
  // images[0] is the cover; the rest are the gallery.
  images: string[];
}

function initialState(product?: Product): FormState {
  const images = [product?.coverImage ?? "", ...(product?.images ?? [])].filter(Boolean);
  return {
    title: product?.title ?? "",
    names: { ...(product?.names ?? {}) },
    description: product?.description ?? "",
    price: product ? (product.priceCents / 100).toFixed(2) : "",
    stock: product ? String(product.stock) : "0",
    weight: product ? String(product.weightGrams ?? 0) : "0",
    status: product?.status ?? "draft",
    categoryId: product?.categoryId ?? "",
    images: [...new Set(images)],
  };
}

export function ProductForm({ product, onDone }: { product?: Product; onDone: () => void }) {
  const { t } = useI18n();
  const { data: categories } = useCategories();
  const create = useCreateProduct();
  const update = useUpdateProduct();
  const upload = useUploadImage();
  const [form, setForm] = useState<FormState>(() => initialState(product));

  const set = (patch: Partial<FormState>) => setForm((f) => ({ ...f, ...patch }));

  async function onFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const res = await upload.mutateAsync(file);
    setForm((f) => ({ ...f, images: [...f.images, res.url] }));
    e.target.value = "";
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    const priceCents = Math.round(Number.parseFloat(form.price || "0") * 100);
    const input = {
      title: form.title,
      names: form.names,
      description: form.description,
      priceCents,
      stock: Number.parseInt(form.stock || "0", 10),
      weightGrams: Number.parseInt(form.weight || "0", 10),
      status: form.status,
      categoryId: form.categoryId || undefined,
      coverImage: form.images[0] ?? "",
      images: form.images.slice(1),
    };

    if (product) {
      update.mutate({ id: product.id, input }, { onSuccess: onDone });
    } else {
      create.mutate(input, { onSuccess: onDone });
    }
  }

  const busy = create.isPending || update.isPending;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{product ? t("ops.editProduct") : t("ops.newProduct")}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="title">{t("ops.title")}</Label>
            <Input
              id="title"
              value={form.title}
              onChange={(e) => set({ title: e.target.value })}
              required
            />
          </div>

          <div className="flex flex-wrap items-end gap-3 sm:col-span-2">
            {supportedLocales.map((locale) => (
              <div key={locale} className="space-y-2">
                <Label htmlFor={`title-${locale}`}>{t("ops.titleIn", { locale: localeNames[locale] })}</Label>
                <Input
                  id={`title-${locale}`}
                  value={form.names[locale] ?? ""}
                  onChange={(e) => set({ names: { ...form.names, [locale]: e.target.value } })}
                  placeholder={form.title}
                />
              </div>
            ))}
          </div>

          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="description">{t("product.description")}</Label>
            <Textarea
              id="description"
              value={form.description}
              onChange={(e) => set({ description: e.target.value })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="price">{t("ops.priceCny")}</Label>
            <Input
              id="price"
              type="number"
              step="0.01"
              min="0"
              value={form.price}
              onChange={(e) => set({ price: e.target.value })}
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="stock">{t("ops.stock")}</Label>
            <Input
              id="stock"
              type="number"
              min="0"
              value={form.stock}
              onChange={(e) => set({ stock: e.target.value })}
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="weight">{t("ops.weightGrams")}</Label>
            <Input
              id="weight"
              type="number"
              min="0"
              value={form.weight}
              onChange={(e) => set({ weight: e.target.value })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="status">{t("common.status")}</Label>
            <select
              id="status"
              value={form.status}
              onChange={(e) => set({ status: e.target.value })}
              className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
            >
              <option value="draft">{t("ops.draft")}</option>
              <option value="published">{t("ops.published")}</option>
              <option value="archived">{t("ops.archived")}</option>
            </select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="category">{t("ops.category")}</Label>
            <select
              id="category"
              value={form.categoryId}
              onChange={(e) => set({ categoryId: e.target.value })}
              className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
            >
              <option value="">{t("ops.uncategorised")}</option>
              {categories?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-2 sm:col-span-2">
            <Label>{t("ops.images")}</Label>
            <div className="flex flex-wrap items-center gap-3">
              {form.images.map((src, index) => (
                <div key={src} className="group relative">
                  <img
                    src={src}
                    alt=""
                    className="bg-muted size-20 rounded-md border object-cover"
                  />
                  {index === 0 && (
                    <span className="bg-primary text-primary-foreground absolute bottom-1 left-1 rounded px-1 text-[10px]">
                      {t("ops.cover")}
                    </span>
                  )}
                  <button
                    type="button"
                    aria-label={t("common.delete")}
                    onClick={() =>
                      setForm((f) => ({ ...f, images: f.images.filter((_, i) => i !== index) }))
                    }
                    className="bg-background absolute -top-1 -right-1 rounded-full border p-0.5 opacity-0 group-hover:opacity-100"
                  >
                    <X className="size-3" />
                  </button>
                </div>
              ))}
              <label className="border-input hover:bg-accent inline-flex h-9 cursor-pointer items-center gap-2 rounded-md border px-3 text-sm">
                <Upload className="size-4" />
                {upload.isPending ? t("ops.uploading") : t("ops.addImage")}
                <input type="file" accept="image/*" className="hidden" onChange={onFile} />
              </label>
            </div>
          </div>

          <div className="flex gap-2 sm:col-span-2">
            <Button type="submit" disabled={busy}>
              {busy ? t("common.saving") : product ? t("ops.saveChanges") : t("ops.createProduct")}
            </Button>
            <Button type="button" variant="outline" onClick={onDone}>
              {t("common.cancel")}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
