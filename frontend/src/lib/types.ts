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
}

export interface CartItem {
  productId: string;
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

export interface Order {
  id: string;
  orderNo: string;
  status: OrderStatus;
  currency: string;
  totalCents: number;
  items: OrderItem[];
  paymentId: string;
  expiresAt: string;
  paidAt?: string;
  createdAt: string;
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
