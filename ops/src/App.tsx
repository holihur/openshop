import { Suspense, lazy } from "react";
import { Navigate, Route, Routes } from "react-router-dom";

import { RouteFallback } from "@lib/components/route-fallback";

import { AdminRoute } from "@/components/admin-route";
import { OpsLayout } from "@/components/OpsLayout";
// Each screen is a separate chunk: the console opens with the shell and the
// dashboard, and the rest arrive on demand.
const DashboardPage = lazy(() => import("@/pages/Dashboard").then((m) => ({ default: m.DashboardPage })));
const ProductsPage = lazy(() => import("@/pages/Products").then((m) => ({ default: m.ProductsPage })));
const ProductNewPage = lazy(() => import("@/pages/ProductNew").then((m) => ({ default: m.ProductNewPage })));
const ProductDetailPage = lazy(() => import("@/pages/ProductDetail").then((m) => ({ default: m.ProductDetailPage })));
const CategoriesPage = lazy(() => import("@/pages/Categories").then((m) => ({ default: m.CategoriesPage })));
const CategoryNewPage = lazy(() => import("@/pages/CategoryNew").then((m) => ({ default: m.CategoryNewPage })));
const OrdersPage = lazy(() => import("@/pages/Orders").then((m) => ({ default: m.OrdersPage })));
const OrderDetailPage = lazy(() => import("@/pages/OrderDetail").then((m) => ({ default: m.OrderDetailPage })));
const CustomersPage = lazy(() => import("@/pages/Customers").then((m) => ({ default: m.CustomersPage })));
const CustomerDetailPage = lazy(() => import("@/pages/CustomerDetail").then((m) => ({ default: m.CustomerDetailPage })));
const CouponsPage = lazy(() => import("@/pages/Coupons").then((m) => ({ default: m.CouponsPage })));
const CouponNewPage = lazy(() => import("@/pages/CouponNew").then((m) => ({ default: m.CouponNewPage })));
const ReviewsPage = lazy(() => import("@/pages/Reviews").then((m) => ({ default: m.ReviewsPage })));
const ReturnsPage = lazy(() => import("@/pages/Returns").then((m) => ({ default: m.ReturnsPage })));
const TicketsPage = lazy(() => import("@/pages/Tickets").then((m) => ({ default: m.TicketsPage })));
const TicketDetailPage = lazy(() => import("@/pages/TicketDetail").then((m) => ({ default: m.TicketDetailPage })));
const CommissionsPage = lazy(() => import("@/pages/Commissions").then((m) => ({ default: m.CommissionsPage })));
const WithdrawalsPage = lazy(() => import("@/pages/Withdrawals").then((m) => ({ default: m.WithdrawalsPage })));
const TokensPage = lazy(() => import("@/pages/Tokens").then((m) => ({ default: m.TokensPage })));
const OutboxPage = lazy(() => import("@/pages/Outbox").then((m) => ({ default: m.OutboxPage })));
const ShippingPage = lazy(() => import("@/pages/Shipping").then((m) => ({ default: m.ShippingPage })));
const AuditPage = lazy(() => import("@/pages/Audit").then((m) => ({ default: m.AuditPage })));
const CurrencyPage = lazy(() => import("@/pages/Currency").then((m) => ({ default: m.CurrencyPage })));
const SettingsPage = lazy(() => import("@/pages/Settings").then((m) => ({ default: m.SettingsPage })));
const StaffPage = lazy(() => import("@/pages/Staff").then((m) => ({ default: m.StaffPage })));
const ReconciliationPage = lazy(() => import("@/pages/Reconciliation").then((m) => ({ default: m.ReconciliationPage })));
const RolesPage = lazy(() => import("@/pages/Roles").then((m) => ({ default: m.RolesPage })));
const RoleNewPage = lazy(() => import("@/pages/RoleDetail").then((m) => ({ default: m.RoleNewPage })));
const OpsLoginPage = lazy(() => import("@/pages/Login").then((m) => ({ default: m.OpsLoginPage })));

export default function App() {
  return (
    <Suspense fallback={<RouteFallback />}>
    <Routes>
      <Route path="login" element={<OpsLoginPage />} />
      <Route element={<AdminRoute />}>
        <Route element={<OpsLayout />}>
          <Route index element={<DashboardPage />} />
          <Route path="products" element={<ProductsPage />} />
          <Route path="products/new" element={<ProductNewPage />} />
          <Route path="products/:id" element={<ProductDetailPage />} />
          <Route path="categories" element={<CategoriesPage />} />
          <Route path="categories/new" element={<CategoryNewPage />} />
          <Route path="orders" element={<OrdersPage />} />
          <Route path="orders/:id" element={<OrderDetailPage />} />
          <Route path="customers" element={<CustomersPage />} />
          <Route path="customers/:id" element={<CustomerDetailPage />} />
          <Route path="coupons" element={<CouponsPage />} />
          <Route path="coupons/new" element={<CouponNewPage />} />
          <Route path="reviews" element={<ReviewsPage />} />
          <Route path="returns" element={<ReturnsPage />} />
          <Route path="tickets" element={<TicketsPage />} />
          <Route path="tickets/:id" element={<TicketDetailPage />} />
          <Route path="commissions" element={<CommissionsPage />} />
          <Route path="withdrawals" element={<WithdrawalsPage />} />
          <Route path="tokens" element={<TokensPage />} />
          <Route path="outbox" element={<OutboxPage />} />
          <Route path="shipping" element={<ShippingPage />} />
          <Route path="audit" element={<AuditPage />} />
          <Route path="reconciliation" element={<ReconciliationPage />} />
          <Route path="staff" element={<StaffPage />} />
          <Route path="roles" element={<RolesPage />} />
          <Route path="roles/new" element={<RoleNewPage />} />
          <Route path="roles/:key" element={<RoleNewPage />} />
          <Route path="settings" element={<SettingsPage />} />
          <Route path="settings/:group" element={<SettingsPage />} />
          <Route path="currency" element={<CurrencyPage />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
    </Suspense>
  );
}
