import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@/lib/api";
import type { Category, Order, Product } from "@/lib/types";

export interface ProductInput {
  categoryId?: string;
  title: string;
  slug?: string;
  description?: string;
  priceCents: number;
  currency?: string;
  coverImage?: string;
  images?: string[];
  status?: string;
  stock: number;
}

export function useCategories() {
  return useQuery({
    queryKey: ["categories"],
    queryFn: () => api.get<Category[]>("/categories"),
    staleTime: 5 * 60_000,
  });
}

export function useAdminProducts(page = 1, pageSize = 100) {
  return useQuery({
    queryKey: ["admin", "products", page, pageSize],
    queryFn: () => api.getPage<Product[]>(`/admin/products?page=${page}&pageSize=${pageSize}`),
  });
}

export function useAdminOrders(page = 1, pageSize = 50) {
  return useQuery({
    queryKey: ["admin", "orders", page, pageSize],
    queryFn: () => api.getPage<Order[]>(`/admin/orders?page=${page}&pageSize=${pageSize}`),
  });
}

function invalidateCatalog(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: ["admin", "products"] });
  void queryClient.invalidateQueries({ queryKey: ["products"] });
  void queryClient.invalidateQueries({ queryKey: ["product"] });
}

export function useCreateProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ProductInput) => api.post<Product>("/admin/products", input),
    onSuccess: () => {
      toast.success("Product created");
      invalidateCatalog(queryClient);
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useUpdateProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<ProductInput> }) =>
      api.patch<Product>(`/admin/products/${id}`, input),
    onSuccess: () => {
      toast.success("Product updated");
      invalidateCatalog(queryClient);
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useUploadImage() {
  return useMutation({
    mutationFn: (file: File) => api.upload<{ key: string; url: string }>("/admin/uploads", file),
    onError: (error: Error) => toast.error(error.message),
  });
}
