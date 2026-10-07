import { Route, Routes } from "react-router-dom";

import { AppLayout } from "@/components/layout/AppLayout";
import { ProtectedRoute } from "@lib/components/protected-route";
import { HomePage } from "@/pages/Home";
import { ProductsPage } from "@/pages/Products";
import { ProductDetailPage } from "@/pages/ProductDetail";
import { CartPage } from "@/pages/Cart";
import { OrdersPage } from "@/pages/Orders";
import { OrderDetailPage } from "@/pages/OrderDetail";
import { LoginPage } from "@/pages/Login";
import { RegisterPage } from "@/pages/Register";
import { ForgotPasswordPage } from "@/pages/ForgotPassword";
import { ResetPasswordPage } from "@/pages/ResetPassword";
import { VerifyEmailPage } from "@/pages/VerifyEmail";
import { GuestOrderPage } from "@/pages/GuestOrder";
import { OidcCallbackPage } from "@/pages/OidcCallback";
import { SupportPage } from "@/pages/Support";
import { TicketDetailPage } from "@/pages/TicketDetail";
import { PaymentResultPage } from "@/pages/PaymentResult";
import { AddressesPage } from "@/pages/Addresses";
import { WishlistPage } from "@/pages/Wishlist";
import { AccountSettingsPage } from "@/pages/AccountSettings";
import { WalletPage } from "@/pages/Wallet";
import { RewardsPage } from "@/pages/Rewards";
import { NotFoundPage } from "@/pages/NotFound";

export default function App() {
  return (
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
        </Route>

        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
