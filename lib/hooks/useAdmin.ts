import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { errorMessage } from "@lib/errors";
import { api } from "@lib/api";
import { t, useI18n } from "@lib/i18n";
import type { AuditLog, Category, Coupon, CouponRedemption, CurrenciesResponse, Customer, Dashboard, ExchangeRate, Order, Product, Review, Setting, Variant } from "@lib/types";

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
  const { locale } = useI18n();
  return useQuery({
    queryKey: ["categories", locale],
    queryFn: () => api.get<Category[]>("/categories"),
    staleTime: 5 * 60_000,
  });
}

export interface CategoryInput {
  name: string;
  names?: Record<string, string>;
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
  /** Unit cost, used for margin reporting; 0 inherits the product cost. */
  costCents?: number;
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

export interface PaymentGatewayStatus {
  name: string;
  /** Whether the settings this gateway reads are filled in. */
  identifiersSet: boolean;
  /** Whether the operator has offered it at checkout. */
  enabled: boolean;
  /** Settings that are still empty — what this page can fix. */
  missing?: string[];
  /** Environment variables the API process must provide. */
  needsEnv?: string[];
}

/** Gateway readiness for the ops console: credentials present and offered. */
export function usePaymentGateways() {
  return useQuery({
    queryKey: ["admin", "payment-gateways"],
    queryFn: () => api.get<PaymentGatewayStatus[]>("/ops/payment-gateways"),
  });
}

export function useSettings() {
  return useQuery({
    queryKey: ["admin", "settings"],
    queryFn: () => api.get<Setting[]>("/ops/settings"),
    staleTime: 0,
  });
}

export function useUpdateSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (values: Record<string, string>) => api.put<Setting[]>("/ops/settings", values),
    onSuccess: (items) => {
      toast.success(t("toast.settingsSaved"));
      queryClient.setQueryData(["admin", "settings"], items);
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useCouponRedemptions(couponId: string) {
  return useQuery({
    queryKey: ["admin", "coupon-redemptions", couponId],
    queryFn: () => api.getPage<CouponRedemption[]>(`/ops/coupons/${couponId}/redemptions?pageSize=100`),
    enabled: Boolean(couponId),
  });
}

export function useAdminCustomers(
  page = 1,
  pageSize = 20,
  filters: { keyword?: string; status?: string } = {},
) {
  return useQuery({
    queryKey: ["admin", "customers", page, pageSize, filters],
    queryFn: () => api.getPage<Customer[]>(`/ops/customers${qs({ page, pageSize, ...filters })}`),
  });
}

export function useAdminCustomer(id: string) {
  return useQuery({
    queryKey: ["admin", "customer", id],
    queryFn: () => api.get<Customer>(`/ops/customers/${id}`),
    enabled: Boolean(id),
  });
}

export function useCustomerOrders(customerId: string) {
  return useQuery({
    queryKey: ["admin", "customer-orders", customerId],
    queryFn: () => api.getPage<Order[]>(`/ops/orders?userId=${customerId}&pageSize=20`),
    enabled: Boolean(customerId),
  });
}

export function useUpdateCustomer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: { name?: string; status?: string } }) =>
      api.patch<Customer>(`/ops/customers/${id}`, input),
    onSuccess: () => {
      toast.success(t("toast.customerUpdated"));
      void queryClient.invalidateQueries({ queryKey: ["admin", "customers"] });
      void queryClient.invalidateQueries({ queryKey: ["admin", "customer"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export interface FAQ {
  id: string;
  question: string;
  answer: string;
}

export function useProductFAQs(productId: string) {
  return useQuery({
    queryKey: ["admin", "product-faqs", productId],
    queryFn: () => api.get<FAQ[]>(`/ops/products/${productId}/faqs`),
    enabled: Boolean(productId),
  });
}

export function useReplaceFAQs(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (faqs: { question: string; answer: string }[]) =>
      api.put<FAQ[]>(`/ops/products/${productId}/faqs`, { faqs }),
    onSuccess: (faqs) => {
      toast.success(t("toast.faqsSaved"));
      queryClient.setQueryData(["admin", "product-faqs", productId], faqs);
      void queryClient.invalidateQueries({ queryKey: ["admin", "product", productId] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useConfirmOrderPayment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post<Order>(`/ops/orders/${id}/confirm-payment`),
    onSuccess: () => {
      toast.success(t("toast.paymentConfirmed"));
      invalidateOrder(queryClient);
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export interface StaffMember {
  id: string;
  email: string;
  name: string;
  role: string;
  status: string;
  createdAt: string;
  permissions: string[];
}

export interface RoleMatrix {
  role: string;
  permissions: string[];
  description: string;
  members: number;
  catalog: string[];
}

export interface StaffInput {
  email: string;
  name?: string;
  role: string;
  password?: string;
}

export interface StaffUpdate {
  name?: string;
  role?: string;
  status?: string;
}

/** Console users only; shoppers live on the customers screen. */
export function useStaff(params: { keyword?: string; role?: string } = {}) {
  const query = new URLSearchParams();
  if (params.keyword) query.set("keyword", params.keyword);
  if (params.role) query.set("role", params.role);
  const suffix = query.toString();
  return useQuery({
    queryKey: ["admin", "staff", suffix],
    queryFn: () => api.getPage<StaffMember[]>(`/ops/staff${suffix ? `?${suffix}` : ""}`),
  });
}

/** The role matrix the middleware enforces, for the roles tab. */
export function useRoleMatrix() {
  return useQuery({
    queryKey: ["admin", "roles"],
    queryFn: () => api.get<RoleMatrix[]>("/ops/roles"),
    staleTime: 5 * 60_000,
  });
}

export function useCreateStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: StaffInput) =>
      api.post<{ staff: StaffMember; generatedPassword?: string }>("/ops/staff", input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "staff"] });
      void queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
    },
  });
}

export function useUpdateStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: StaffUpdate }) =>
      api.patch<StaffMember>(`/ops/staff/${id}`, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "staff"] });
      void queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
    },
  });
}

export function useResetStaffPassword() {
  return useMutation({
    mutationFn: (id: string) =>
      api.post<{ staff: StaffMember; generatedPassword?: string }>(`/ops/staff/${id}/password`),
  });
}

export interface ReconciliationReport {
  clean: boolean;
  mismatches: number;
  walletDrift?: { id: string; ownerId: string; unit: string; stored: number; expected: number }[];
  pointsDrift?: { id: string; ownerId: string; unit: string; stored: number; expected: number }[];
  orderDrift?: { id: string; orderNo: string; expectedCents: number; actualCents: number }[];
}

/** The money invariants, checked on read so the console shows live state. */
export function useReconciliation() {
  return useQuery({
    queryKey: ["admin", "reconciliation"],
    queryFn: () => api.get<ReconciliationReport>("/ops/reconciliation"),
    staleTime: 30_000,
  });
}
