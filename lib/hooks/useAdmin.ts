import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { errorMessage } from "@lib/errors";
import { api } from "@lib/api";
import { t } from "@lib/i18n";
import type { AuditLog, Category, Coupon, CurrenciesResponse, Dashboard, ExchangeRate, Order, Product, Review, Variant } from "@lib/types";

function qs(params: Record<string, string | number | undefined>): string {
  const sp = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== "") sp.set(key, String(value));
  }
  const s = sp.toString();
  return s ? `?${s}` : "";
}

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

export interface CategoryInput {
  name: string;
  slug?: string;
  parentId?: string;
  sort?: number;
}

export function useCreateCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CategoryInput) => api.post<Category>("/ops/categories", input),
    onSuccess: () => {
      toast.success(t("toast.categoryCreated"));
      void queryClient.invalidateQueries({ queryKey: ["categories"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useUpdateCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<CategoryInput> }) =>
      api.patch<Category>(`/ops/categories/${id}`, input),
    onSuccess: () => {
      toast.success(t("toast.categoryUpdated"));
      void queryClient.invalidateQueries({ queryKey: ["categories"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useAdminProducts(
  page = 1,
  pageSize = 20,
  filters: { keyword?: string; categoryId?: string; sort?: string } = {},
) {
  return useQuery({
    queryKey: ["admin", "products", page, pageSize, filters],
    queryFn: () => api.getPage<Product[]>(`/ops/products${qs({ page, pageSize, ...filters })}`),
  });
}

/** One product with its variants, for the ops detail page. */
export function useAdminProduct(id: string) {
  return useQuery({
    queryKey: ["admin", "product", id],
    queryFn: () => api.get<Product>(`/ops/products/${id}`),
    enabled: Boolean(id),
  });
}

export function useAdminOrders(page = 1, pageSize = 20, status?: string) {
  return useQuery({
    queryKey: ["admin", "orders", page, pageSize, status],
    queryFn: () => api.getPage<Order[]>(`/ops/orders${qs({ page, pageSize, status })}`),
  });
}

/** One order, for the ops order-detail page. */
export function useAdminOrder(id: string) {
  return useQuery({
    queryKey: ["admin", "order", id],
    queryFn: () => api.get<Order>(`/ops/orders/${id}`),
    enabled: Boolean(id),
  });
}

function invalidateOrder(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: ["admin", "orders"] });
  void queryClient.invalidateQueries({ queryKey: ["admin", "order"] });
}

export function useShipOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, trackingNo }: { id: string; trackingNo: string }) =>
      api.post<Order>(`/ops/orders/${id}/ship`, { trackingNo }),
    onSuccess: () => {
      toast.success(t("ops.orderShipped"));
      invalidateOrder(queryClient);
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useCompleteOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post<Order>(`/ops/orders/${id}/complete`),
    onSuccess: () => {
      toast.success(t("ops.orderCompleted"));
      invalidateOrder(queryClient);
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useRefundOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      amountCents,
      restock,
      reason,
    }: {
      id: string;
      amountCents: number;
      restock: boolean;
      reason?: string;
    }) => api.post<Order>(`/ops/orders/${id}/refund`, { reason: reason ?? "admin refund", amountCents, restock }),
    onSuccess: () => {
      toast.success(t("ops.orderRefunded"));
      invalidateOrder(queryClient);
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useDashboard() {
  return useQuery({
    queryKey: ["admin", "dashboard"],
    queryFn: () => api.get<Dashboard>("/ops/stats"),
  });
}

export function useAuditLogs(page = 1, pageSize = 20, action?: string) {
  return useQuery({
    queryKey: ["admin", "audit", page, pageSize, action],
    queryFn: () => api.getPage<AuditLog[]>(`/ops/audit-logs${qs({ page, pageSize, action })}`),
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
      toast.success(t("toast.exchangeRateSaved"));
      void queryClient.invalidateQueries({ queryKey: ["currencies"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useAdminCoupons(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ["admin", "coupons", page, pageSize],
    queryFn: () => api.getPage<Coupon[]>(`/ops/coupons${qs({ page, pageSize })}`),
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
      toast.success(t("toast.couponCreated"));
      void queryClient.invalidateQueries({ queryKey: ["admin", "coupons"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useUpdateCoupon() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<CouponInput> }) =>
      api.patch<Coupon>(`/ops/coupons/${id}`, input),
    onSuccess: () => {
      toast.success(t("toast.couponUpdated"));
      void queryClient.invalidateQueries({ queryKey: ["admin", "coupons"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useAdminReviews(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ["admin", "reviews", page, pageSize],
    queryFn: () => api.getPage<Review[]>(`/ops/reviews${qs({ page, pageSize })}`),
  });
}

export function useDeleteReviewAdmin() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.del(`/ops/reviews/${id}`),
    onSuccess: () => {
      toast.success(t("toast.reviewRemoved"));
      void queryClient.invalidateQueries({ queryKey: ["admin", "reviews"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

function invalidateCatalog(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: ["admin", "products"] });
  void queryClient.invalidateQueries({ queryKey: ["admin", "product"] });
  void queryClient.invalidateQueries({ queryKey: ["products"] });
  void queryClient.invalidateQueries({ queryKey: ["product"] });
}

export function useCreateProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ProductInput) => api.post<Product>("/ops/products", input),
    onSuccess: () => {
      toast.success(t("toast.productCreated"));
      invalidateCatalog(queryClient);
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useUpdateProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<ProductInput> }) =>
      api.patch<Product>(`/ops/products/${id}`, input),
    onSuccess: () => {
      toast.success(t("toast.productUpdated"));
      invalidateCatalog(queryClient);
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useUploadImage() {
  return useMutation({
    mutationFn: (file: File) => api.upload<{ key: string; url: string }>("/ops/uploads", file),
    onError: (error: Error) => toast.error(errorMessage(error)),
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
      toast.success(t("toast.variantAdded"));
      invalidateVariants(queryClient, productId);
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
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
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}
