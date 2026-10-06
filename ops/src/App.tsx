import { Navigate, Route, Routes } from "react-router-dom";

import { AdminRoute } from "@/components/admin-route";
import { OpsLayout } from "@/components/OpsLayout";
import { AdminPage } from "@/pages/AdminPage";
import { OpsLoginPage } from "@/pages/Login";

export default function App() {
  return (
    <Routes>
      <Route path="login" element={<OpsLoginPage />} />
      <Route element={<AdminRoute />}>
        <Route element={<OpsLayout />}>
          <Route index element={<AdminPage />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
