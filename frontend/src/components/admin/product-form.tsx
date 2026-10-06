import { useState, type ChangeEvent, type FormEvent } from "react";
import { Upload } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useCategories, useCreateProduct, useUpdateProduct, useUploadImage } from "@/hooks/useAdmin";
import type { Product } from "@/lib/types";

interface FormState {
  title: string;
  description: string;
  price: string; // in major units, converted to cents on submit
  stock: string;
  status: string;
  categoryId: string;
  coverImage: string;
}

function initialState(product?: Product): FormState {
  return {
    title: product?.title ?? "",
    description: product?.description ?? "",
    price: product ? (product.priceCents / 100).toFixed(2) : "",
    stock: product ? String(product.stock) : "0",
    status: product?.status ?? "draft",
    categoryId: product?.categoryId ?? "",
    coverImage: product?.coverImage ?? "",
  };
}

export function ProductForm({ product, onDone }: { product?: Product; onDone: () => void }) {
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
    set({ coverImage: res.url });
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    const priceCents = Math.round(Number.parseFloat(form.price || "0") * 100);
    const input = {
      title: form.title,
      description: form.description,
      priceCents,
      stock: Number.parseInt(form.stock || "0", 10),
      status: form.status,
      categoryId: form.categoryId || undefined,
      coverImage: form.coverImage,
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
        <CardTitle>{product ? "Edit product" : "New product"}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="title">Title</Label>
            <Input
              id="title"
              value={form.title}
              onChange={(e) => set({ title: e.target.value })}
              required
            />
          </div>

          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={form.description}
              onChange={(e) => set({ description: e.target.value })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="price">Price (CNY)</Label>
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
            <Label htmlFor="stock">Stock</Label>
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
            <Label htmlFor="status">Status</Label>
            <select
              id="status"
              value={form.status}
              onChange={(e) => set({ status: e.target.value })}
              className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
            >
              <option value="draft">Draft</option>
              <option value="published">Published</option>
              <option value="archived">Archived</option>
            </select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="category">Category</Label>
            <select
              id="category"
              value={form.categoryId}
              onChange={(e) => set({ categoryId: e.target.value })}
              className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
            >
              <option value="">Uncategorised</option>
              {categories?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-2 sm:col-span-2">
            <Label>Cover image</Label>
            <div className="flex items-center gap-3">
              {form.coverImage && (
                <img
                  src={form.coverImage}
                  alt="cover"
                  className="bg-muted size-16 rounded-md border object-cover"
                />
              )}
              <label className="border-input hover:bg-accent inline-flex h-9 cursor-pointer items-center gap-2 rounded-md border px-3 text-sm">
                <Upload className="size-4" />
                {upload.isPending ? "Uploading…" : "Upload image"}
                <input type="file" accept="image/*" className="hidden" onChange={onFile} />
              </label>
            </div>
          </div>

          <div className="flex gap-2 sm:col-span-2">
            <Button type="submit" disabled={busy}>
              {busy ? "Saving…" : product ? "Save changes" : "Create product"}
            </Button>
            <Button type="button" variant="outline" onClick={onDone}>
              Cancel
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
