import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import type { AuditLog, Category, Coupon, CurrenciesResponse, Dashboard, ExchangeRate, Order, Product, Review, Variant } from "@lib/types";

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
  weightGrams?: number;
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
    queryFn: () => api.getPage<Product[]>(`/ops/products?page=${page}&pageSize=${pageSize}`),
  });
}

export function useAdminOrders(page = 1, pageSize = 50) {
  return useQuery({
    queryKey: ["admin", "orders", page, pageSize],
    queryFn: () => api.getPage<Order[]>(`/ops/orders?page=${page}&pageSize=${pageSize}`),
  });
}

export function useDashboard() {
  return useQuery({
    queryKey: ["admin", "dashboard"],
    queryFn: () => api.get<Dashboard>("/ops/stats"),
  });
}

export function useAuditLogs() {
  return useQuery({
    queryKey: ["admin", "audit"],
    queryFn: () => api.getPage<AuditLog[]>("/ops/audit-logs?pageSize=100"),
  });
}

export function useCurrencies() {
  return useQuery({
    queryKey: ["currencies"],
    queryFn: () => api.get<CurrenciesResponse>("/currencies"),
  });
}

export function useSetCurrencyRate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ code, rateMicro }: { code: string; rateMicro: number }) =>
      api.put<ExchangeRate>(`/ops/currencies/${code}`, { rateMicro }),
    onSuccess: () => {
      toast.success("Exchange rate saved");
      void queryClient.invalidateQueries({ queryKey: ["currencies"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useAdminCoupons() {
  return useQuery({
    queryKey: ["admin", "coupons"],
    queryFn: () => api.get<Coupon[]>("/ops/coupons"),
  });
}

export interface CouponInput {
  code: string;
  description?: string;
  discountType: "percent" | "fixed";
  discountValue: number;
  minSubtotalCents?: number;
  maxDiscountCents?: number;
  usageLimit?: number;
  perUserLimit?: number;
  active?: boolean;
}

export function useCreateCoupon() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CouponInput) => api.post<Coupon>("/ops/coupons", input),
    onSuccess: () => {
      toast.success("Coupon created");
      void queryClient.invalidateQueries({ queryKey: ["admin", "coupons"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useUpdateCoupon() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<CouponInput> }) =>
      api.patch<Coupon>(`/ops/coupons/${id}`, input),
    onSuccess: () => {
      toast.success("Coupon updated");
      void queryClient.invalidateQueries({ queryKey: ["admin", "coupons"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useAdminReviews() {
  return useQuery({
    queryKey: ["admin", "reviews"],
    queryFn: () => api.getPage<Review[]>("/ops/reviews?pageSize=100"),
  });
}

export function useDeleteReviewAdmin() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.del(`/ops/reviews/${id}`),
    onSuccess: () => {
      toast.success("Review removed");
      void queryClient.invalidateQueries({ queryKey: ["admin", "reviews"] });
    },
    onError: (error: Error) => toast.error(error.message),
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
    mutationFn: (input: ProductInput) => api.post<Product>("/ops/products", input),
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
      api.patch<Product>(`/ops/products/${id}`, input),
    onSuccess: () => {
      toast.success("Product updated");
      invalidateCatalog(queryClient);
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useUploadImage() {
  return useMutation({
    mutationFn: (file: File) => api.upload<{ key: string; url: string }>("/ops/uploads", file),
    onError: (error: Error) => toast.error(error.message),
  });
}

export interface VariantInput {
  sku?: string;
  name: string;
  priceCents: number;
  stock: number;
  weightGrams?: number;
  active?: boolean;
}

export function useVariants(productId: string) {
  return useQuery({
    queryKey: ["admin", "variants", productId],
    queryFn: () => api.get<Variant[]>(`/ops/products/${productId}/variants`),
    enabled: Boolean(productId),
  });
}

function invalidateVariants(queryClient: ReturnType<typeof useQueryClient>, productId: string) {
  void queryClient.invalidateQueries({ queryKey: ["admin", "variants", productId] });
  void queryClient.invalidateQueries({ queryKey: ["product", productId] });
}

export function useCreateVariant(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: VariantInput) =>
      api.post<Variant>(`/ops/products/${productId}/variants`, input),
    onSuccess: () => {
      toast.success("Variant added");
      invalidateVariants(queryClient, productId);
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useUpdateVariant(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<VariantInput> }) =>
      api.patch<Variant>(`/ops/variants/${id}`, input),
    onSuccess: () => {
      invalidateVariants(queryClient, productId);
    },
    onError: (error: Error) => toast.error(error.message),
  });
}
