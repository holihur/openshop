export interface User {
  id: string;
  email: string;
  phone: string;
  name: string;
  role: "customer" | "admin";
}

export interface Category {
  id: string;
  name: string;
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
  variants?: Variant[];
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
  totalCents: number;
  items: OrderItem[];
  paymentId: string;
  shippingAddress?: Address;
  trackingNo?: string;
  shippedAt?: string;
  completedAt?: string;
  expiresAt: string;
  paidAt?: string;
  createdAt: string;
}

export interface Review {
  id: string;
  productId: string;
  userId: string;
  rating: number;
  title: string;
  body: string;
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

export interface Dashboard {
  revenueCents: number;
  paidOrders: number;
  pendingOrders: number;
  cancelledOrders: number;
  totalOrders: number;
  totalProducts: number;
  totalUsers: number;
  recentOrders: Order[];
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
