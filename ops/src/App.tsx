import { Navigate, Route, Routes } from "react-router-dom";

import { AdminRoute } from "@/components/admin-route";
import { OpsLayout } from "@/components/OpsLayout";
import { DashboardPage } from "@/pages/Dashboard";
import { ProductsPage } from "@/pages/Products";
import { OrdersPage } from "@/pages/Orders";
import { OrderDetailPage } from "@/pages/OrderDetail";
import { ProductDetailPage } from "@/pages/ProductDetail";
import { CategoriesPage } from "@/pages/Categories";
import { SettingsPage } from "@/pages/Settings";
import { CouponsPage } from "@/pages/Coupons";
import { ReviewsPage } from "@/pages/Reviews";
import { ReturnsPage } from "@/pages/Returns";
import { TicketsPage } from "@/pages/Tickets";
import { TicketDetailPage } from "@/pages/TicketDetail";
import { CommissionsPage } from "@/pages/Commissions";
import { WithdrawalsPage } from "@/pages/Withdrawals";
import { TokensPage } from "@/pages/Tokens";
import { CustomersPage } from "@/pages/Customers";
import { CustomerDetailPage } from "@/pages/CustomerDetail";
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
          <Route path="products/:id" element={<ProductDetailPage />} />
          <Route path="categories" element={<CategoriesPage />} />
          <Route path="orders" element={<OrdersPage />} />
          <Route path="orders/:id" element={<OrderDetailPage />} />
          <Route path="customers" element={<CustomersPage />} />
          <Route path="customers/:id" element={<CustomerDetailPage />} />
          <Route path="coupons" element={<CouponsPage />} />
          <Route path="reviews" element={<ReviewsPage />} />
          <Route path="returns" element={<ReturnsPage />} />
          <Route path="tickets" element={<TicketsPage />} />
          <Route path="tickets/:id" element={<TicketDetailPage />} />
          <Route path="commissions" element={<CommissionsPage />} />
          <Route path="withdrawals" element={<WithdrawalsPage />} />
          <Route path="tokens" element={<TokensPage />} />
          <Route path="shipping" element={<ShippingPage />} />
          <Route path="audit" element={<AuditPage />} />
          <Route path="settings" element={<SettingsPage />} />
          <Route path="settings/:group" element={<SettingsPage />} />
          <Route path="currency" element={<CurrencyPage />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
