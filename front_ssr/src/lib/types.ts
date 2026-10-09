/**
 * The API contracts this storefront consumes. They mirror the Go views; the
 * server-rendered app deliberately keeps its own copy so it can be built and
 * deployed independently of the Vite storefront.
 */

export interface SiteHero {
  title: string;
  subtitle: string;
  image: string;
  ctaUrl: string;
}

export interface SiteFeature {
  title: string;
  text: string;
}

export interface SiteConfig {
  publicUrl: string;
  hero: SiteHero;
  announcement: { title: string; text: string };
  features: SiteFeature[];
  tagline: string;
  themeColor: string;
  allowRegistration: boolean;
  oidcEnabled: boolean;
  oidcProviders: { id: string; name: string }[];
  /** WeChat/Alipay sign-in buttons; an unusable provider is omitted. */
  socialProviders?: { id: string; name: string }[];
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
  active: boolean;
}

export interface Faq {
  id: string;
  question: string;
  answer: string;
}

export interface Product {
  id: string;
  categoryId: string;
  title: string;
  names?: Record<string, string>;
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
  faqs?: Faq[];
  rating?: number;
  reviewCount?: number;
}

export interface ProductFacets {
  minPriceCents: number;
  maxPriceCents: number;
  attributes: { name: string; values: string[] }[];
}

export interface DeliveryEstimate {
  methodId: string;
  code: string;
  name: string;
  priceCents: number;
  minDays: number;
  maxDays: number;
  earliest: string;
  latest: string;
  freeThresholdCents: number;
  freeRemainingCents: number;
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

export interface AuthSession {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: { id: string; email: string; name: string; role: string };
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
}

export interface Order {
  id: string;
  orderNo: string;
  status: string;
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
  createdAt: string;
}

export type PaymentMethod = string;

export interface Address {
  id: string;
  recipient: string;
  phone: string;
  province: string;
  city: string;
  line1: string;
  line2?: string;
  postalCode?: string;
  default?: boolean;
}

export interface ShippingMethod {
  id: string;
  code: string;
  name: string;
  flatRateCents: number;
  freeThresholdCents: number;
  minDays: number;
  maxDays: number;
  active: boolean;
  sort: number;
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
