import { Suspense, lazy } from "react";
import { Route, Routes } from "react-router-dom";

import { AppLayout } from "@/components/layout/AppLayout";
import { ProtectedRoute } from "@lib/components/protected-route";
import { RouteFallback } from "@lib/components/route-fallback";
import { HomePage } from "@/pages/Home";
import { ProductsPage } from "@/pages/Products";
import { ProductDetailPage } from "@/pages/ProductDetail";

// Screens are separate chunks so the storefront opens with the shell and the
// catalog, and the account or checkout code arrives when it is needed.
const CartPage = lazy(() => import("@/pages/Cart").then((m) => ({ default: m.CartPage })));
const OrdersPage = lazy(() => import("@/pages/Orders").then((m) => ({ default: m.OrdersPage })));
const OrderDetailPage = lazy(() => import("@/pages/OrderDetail").then((m) => ({ default: m.OrderDetailPage })));
const LoginPage = lazy(() => import("@/pages/Login").then((m) => ({ default: m.LoginPage })));
const RegisterPage = lazy(() => import("@/pages/Register").then((m) => ({ default: m.RegisterPage })));
const ForgotPasswordPage = lazy(() => import("@/pages/ForgotPassword").then((m) => ({ default: m.ForgotPasswordPage })));
const ResetPasswordPage = lazy(() => import("@/pages/ResetPassword").then((m) => ({ default: m.ResetPasswordPage })));
const VerifyEmailPage = lazy(() => import("@/pages/VerifyEmail").then((m) => ({ default: m.VerifyEmailPage })));
const GuestOrderPage = lazy(() => import("@/pages/GuestOrder").then((m) => ({ default: m.GuestOrderPage })));
const OidcCallbackPage = lazy(() => import("@/pages/OidcCallback").then((m) => ({ default: m.OidcCallbackPage })));
const SupportPage = lazy(() => import("@/pages/Support").then((m) => ({ default: m.SupportPage })));
const TicketDetailPage = lazy(() => import("@/pages/TicketDetail").then((m) => ({ default: m.TicketDetailPage })));
const PaymentResultPage = lazy(() => import("@/pages/PaymentResult").then((m) => ({ default: m.PaymentResultPage })));
const AddressesPage = lazy(() => import("@/pages/Addresses").then((m) => ({ default: m.AddressesPage })));
const WishlistPage = lazy(() => import("@/pages/Wishlist").then((m) => ({ default: m.WishlistPage })));
const AccountSettingsPage = lazy(() => import("@/pages/AccountSettings").then((m) => ({ default: m.AccountSettingsPage })));
const WalletPage = lazy(() => import("@/pages/Wallet").then((m) => ({ default: m.WalletPage })));
const RewardsPage = lazy(() => import("@/pages/Rewards").then((m) => ({ default: m.RewardsPage })));
const NotificationsPage = lazy(() => import("@/pages/Notifications").then((m) => ({ default: m.NotificationsPage })));
const AccountTokensPage = lazy(() => import("@/pages/AccountTokens").then((m) => ({ default: m.AccountTokensPage })));
const NotFoundPage = lazy(() => import("@/pages/NotFound").then((m) => ({ default: m.NotFoundPage })));

export default function App() {
  return (
    <Suspense fallback={<RouteFallback />}>
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<HomePage />} />
        <Route path="products" element={<ProductsPage />} />
        <Route path="products/:id" element={<ProductDetailPage />} />
        <Route path="login" element={<LoginPage />} />
        <Route path="register" element={<RegisterPage />} />
        <Route path="forgot-password" element={<ForgotPasswordPage />} />
        <Route path="reset-password" element={<ResetPasswordPage />} />
        <Route path="verify-email" element={<VerifyEmailPage />} />
        <Route path="oidc/callback" element={<OidcCallbackPage />} />
        <Route path="guest/orders/:token" element={<GuestOrderPage />} />
        <Route path="payment/result" element={<PaymentResultPage />} />
        {/* The cart is public: anonymous shoppers can add items and check out
            as a guest, so it must not sit behind the auth guard. */}
        <Route path="cart" element={<CartPage />} />
        {/* Support is public: guests may open a ticket from the contact form. */}
        <Route path="support" element={<SupportPage />} />

        <Route element={<ProtectedRoute />}>
          <Route path="orders" element={<OrdersPage />} />
          <Route path="orders/:id" element={<OrderDetailPage />} />
          <Route path="support/:id" element={<TicketDetailPage />} />
          <Route path="account/addresses" element={<AddressesPage />} />
          <Route path="account/wishlist" element={<WishlistPage />} />
          <Route path="account/settings" element={<AccountSettingsPage />} />
          <Route path="account/wallet" element={<WalletPage />} />
          <Route path="account/rewards" element={<RewardsPage />} />
          <Route path="account/notifications" element={<NotificationsPage />} />
          <Route path="account/tokens" element={<AccountTokensPage />} />
        </Route>

        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
    </Suspense>
  );
}
