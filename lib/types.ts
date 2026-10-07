export type UserRole = "customer" | "admin" | "support" | "catalog" | "finance";
export interface User {
  id: string;
  email: string;
  phone: string;
  name: string;
  role: UserRole;
  emailVerified: boolean;
  /** Fine-grained ops capabilities; present for ops roles only. */
  permissions?: string[];
}

export interface Setting {
  key: string;
  group: string;
  type: "string" | "int" | "bool" | "color" | "json";
  value: string;
  default: string;
  description: string;
  min?: number;
  max?: number;
}

export interface Site {
  publicUrl: string;
  hero: { title: string; subtitle: string; image: string; ctaUrl: string };
  announcement: { message: string; url: string };
  features: { title: string; text: string }[];
  tagline: string;
  themeColor: string;
  allowRegistration: boolean;
  oidcEnabled: boolean;
  oidcProviders: { id: string; name: string }[];
  withdrawalEnabled: boolean;
  withdrawalMinCents: number;
  withdrawalInstructions: string;
}

export interface Category {
  id: string;
  name: string;
  names?: Record<string, string>;
  slug: string;
  parentId: string;
  sort: number;
}

export interface Variant {
  id: string;
  sku: string;
  name: string;
  priceCents: number;
  stock: number;
  weightGrams: number;
  attributes?: Record<string, string>;
  sort: number;
  active: boolean;
}

export interface Product {
  id: string;
  categoryId: string;
  title: string;
  slug: string;
  description: string;
  priceCents: number;
  currency: string;
  coverImage: string;
  images: string[];
  status: "draft" | "published" | "archived";
  stock: number;
  weightGrams: number;
  variants?: Variant[];
  faqs?: { id: string; question: string; answer: string }[];
  rating?: number;
  reviewCount?: number;
}

export interface CartItem {
  productId: string;
  variantId?: string;
  variantName?: string;
  title: string;
  coverImage: string;
  priceCents: number;
  currency: string;
  quantity: number;
}

export interface Cart {
  items: CartItem[];
  totalCents: number;
  totalCount: number;
}

export interface OrderItem {
  id: string;
  productId: string;
  variantId?: string;
  variantName?: string;
  sku?: string;
  title: string;
  priceCents: number;
  quantity: number;
  subtotal: number;
}

export type OrderStatus =
  | "pending_payment"
  | "paid"
  | "cancelled"
  | "shipped"
  | "completed"
  | "refunded";

export interface Address {
  id: string;
  recipient: string;
  phone: string;
  province: string;
  city: string;
  district: string;
  line1: string;
  postalCode: string;
  default: boolean;
}

export interface Order {
  id: string;
  orderNo: string;
  status: OrderStatus;
  currency: string;
  subtotalCents: number;
  discountCents: number;
  couponCode?: string;
  shippingCents: number;
  taxCents: number;
  shippingMethod?: string;
  totalCents: number;
  refundedCents: number;
  items: OrderItem[];
  paymentId: string;
  shippingAddress?: Address;
  trackingNo?: string;
  shippedAt?: string;
  completedAt?: string;
  expiresAt: string;
  paidAt?: string;
  createdAt: string;
  accessToken?: string;
}

export interface ReturnRequest {
  id: string;
  orderId: string;
  userId: string;
  reason: string;
  status: "requested" | "approved" | "rejected";
  createdAt: string;
}

export interface Wallet {
  currency: string;
  balanceCents: number;
}

export type WalletTransactionType =
  | "topup"
  | "purchase"
  | "refund"
  | "commission"
  | "withdrawal"
  | "adjustment";

export interface WalletTransaction {
  id: string;
  type: WalletTransactionType;
  amountCents: number;
  balanceAfter: number;
  referenceType?: string;
  referenceId?: string;
  description: string;
  createdAt: string;
}

export interface PointsAccount {
  balance: number;
  lifetimeEarned: number;
}

export interface PointsTransaction {
  id: string;
  type: string;
  points: number;
  balanceAfter: number;
  description: string;
  createdAt: string;
}

export type CommissionStatus = "pending" | "approved" | "reversed";

export interface Commission {
  id: string;
  referrerId: string;
  refereeId: string;
  orderId: string;
  baseCents: number;
  rateBps: number;
  amountCents: number;
  status: CommissionStatus;
  holdUntil: string;
  approvedAt?: string;
  createdAt: string;
}

export interface ReferralSummary {
  code: string;
  referrals: number;
  pendingCents: number;
  approvedCents: number;
}

export interface PersonalAccessToken {
  id: string;
  name: string;
  prefix: string;
  realm: string;
  scopes: string[];
  cidrs: string[];
  expiresAt?: string;
  lastUsedAt?: string;
  revokedAt?: string;
  createdAt: string;
}

export interface CreatedPersonalAccessToken extends PersonalAccessToken {
  token: string;
}

export interface ScopeInfo {
  scope: string;
  group: string;
  description: string;
}

export type NotificationType =
  | "system"
  | "order"
  | "ticket"
  | "wallet"
  | "commission"
  | "withdrawal";

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  body?: string;
  link?: string;
  read: boolean;
  readAt?: string;
  createdAt: string;
  data?: Record<string, unknown>;
}

export type WithdrawalStatus = "requested" | "approved" | "paid" | "rejected" | "cancelled";
export type WithdrawalMethod = "bank" | "alipay" | "wechat" | "other";

export interface Withdrawal {
  id: string;
  userId: string;
  amountCents: number;
  currency: string;
  method: WithdrawalMethod;
  accountName: string;
  accountNo: string;
  note?: string;
  status: WithdrawalStatus;
  rejectReason?: string;
  paidReference?: string;
  reviewedAt?: string;
  paidAt?: string;
  createdAt: string;
  updatedAt: string;
}

export type TicketKind = "presale" | "postsale" | "other";
export type TicketStatus = "open" | "pending" | "resolved" | "closed";
export type TicketPriority = "low" | "normal" | "high" | "urgent";

export interface Ticket {
  id: string;
  number: number;
  reference: string;
  userId?: string;
  email: string;
  name: string;
  subject: string;
  kind: TicketKind;
  priority: TicketPriority;
  status: TicketStatus;
  orderId?: string;
  productId?: string;
  assigneeId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface TicketMessage {
  id: string;
  authorId?: string;
  authorRole: "customer" | "staff" | "system";
  authorName?: string;
  body: string;
  internal: boolean;
  createdAt: string;
}

export interface Review {
  id: string;
  productId: string;
  userId: string;
  rating: number;
  title: string;
  body: string;
  verifiedPurchase: boolean;
  createdAt: string;
}

export interface CouponPreview {
  code: string;
  discountCents: number;
  totalCents: number;
}

export interface Coupon {
  id: string;
  code: string;
  description: string;
  discountType: "percent" | "fixed";
  discountValue: number;
  minSubtotalCents: number;
  maxDiscountCents: number;
  usageLimit: number;
  usedCount: number;
  perUserLimit: number;
  active: boolean;
}

export interface CouponRedemption {
  id: string;
  orderId: string;
  orderNo: string;
  userId: string;
  userEmail: string;
  discountCents: number;
  createdAt: string;
}

export interface ShippingMethod {
  id: string;
  code: string;
  name: string;
  flatRateCents: number;
  freeThresholdCents: number;
  active: boolean;
  sort: number;
}

export interface ShippingZone {
  id: string;
  name: string;
  provinces: string[];
  active: boolean;
  sort: number;
}

export interface AuditLog {
  id: string;
  actorId: string;
  actorRole: string;
  action: string;
  resourceType: string;
  resourceId: string;
  metadata?: Record<string, string>;
  ip: string;
  createdAt: string;
}

export interface ExchangeRate {
  currency: string;
  rateMicro: number;
  updatedAt: string;
}

export interface CurrenciesResponse {
  base: string;
  rates: ExchangeRate[];
}

export interface Dashboard {
  revenueCents: number;
  revenueByCurrency?: Record<string, number>;
  lowStock: LowStockItem[];
  paidOrders: number;
  pendingOrders: number;
  cancelledOrders: number;
  totalOrders: number;
  totalProducts: number;
  totalUsers: number;
  recentOrders: Order[];
}

export interface LowStockItem {
  type: "product" | "variant";
  id: string;
  productId: string;
  title: string;
  variantName?: string;
  sku?: string;
  stock: number;
}

export interface Payment {
  id: string;
  orderId: string;
  provider: string;
  status: string;
  amountCents: number;
  currency: string;
  redirectUrl?: string;
}

export interface AuthResponse {
  user: User;
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

export interface Customer {
  id: string;
  email: string;
  phone: string;
  name: string;
  role: string;
  status: "active" | "disabled";
  emailVerified: boolean;
  createdAt: string;
}
