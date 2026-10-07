import { Navigate, Route, Routes } from "react-router-dom";

import { AdminRoute } from "@/components/admin-route";
import { OpsLayout } from "@/components/OpsLayout";
import { DashboardPage } from "@/pages/Dashboard";
import { ProductsPage } from "@/pages/Products";
import { OrdersPage } from "@/pages/Orders";
import { CouponsPage } from "@/pages/Coupons";
import { ReviewsPage } from "@/pages/Reviews";
import { ReturnsPage } from "@/pages/Returns";
import { ShippingPage } from "@/pages/Shipping";
import { AuditPage } from "@/pages/Audit";
import { CurrencyPage } from "@/pages/Currency";
import { OpsLoginPage } from "@/pages/Login";

export default function App() {
  return (
    <Routes>
      <Route path="login" element={<OpsLoginPage />} />
      <Route element={<AdminRoute />}>
        <Route element={<OpsLayout />}>
          <Route index element={<DashboardPage />} />
          <Route path="products" element={<ProductsPage />} />
          <Route path="orders" element={<OrdersPage />} />
          <Route path="coupons" element={<CouponsPage />} />
          <Route path="reviews" element={<ReviewsPage />} />
          <Route path="returns" element={<ReturnsPage />} />
          <Route path="shipping" element={<ShippingPage />} />
          <Route path="audit" element={<AuditPage />} />
          <Route path="currency" element={<CurrencyPage />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
