import { Route, Routes } from "react-router-dom";

import { AppLayout } from "@/components/layout/AppLayout";
import { ProtectedRoute } from "@/components/protected-route";
import { AdminRoute } from "@/components/admin-route";
import { HomePage } from "@/pages/Home";
import { ProductsPage } from "@/pages/Products";
import { ProductDetailPage } from "@/pages/ProductDetail";
import { CartPage } from "@/pages/Cart";
import { OrdersPage } from "@/pages/Orders";
import { OrderDetailPage } from "@/pages/OrderDetail";
import { LoginPage } from "@/pages/Login";
import { RegisterPage } from "@/pages/Register";
import { PaymentResultPage } from "@/pages/PaymentResult";
import { AddressesPage } from "@/pages/Addresses";
import { AdminPage } from "@/pages/admin/AdminPage";
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
        <Route path="payment/result" element={<PaymentResultPage />} />

        <Route element={<ProtectedRoute />}>
          <Route path="cart" element={<CartPage />} />
          <Route path="orders" element={<OrdersPage />} />
          <Route path="orders/:id" element={<OrderDetailPage />} />
          <Route path="account/addresses" element={<AddressesPage />} />
        </Route>

        <Route element={<AdminRoute />}>
          <Route path="admin" element={<AdminPage />} />
        </Route>

        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
