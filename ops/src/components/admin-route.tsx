import { Navigate, Outlet, useLocation } from "react-router-dom";

import { Skeleton } from "@lib/components/ui/skeleton";
import { useAuth } from "@lib/auth";
import type { UserRole } from "@lib/types";

// Every role allowed to sign in to the ops console. Per-section access is
// enforced by the API and reflected in the sidebar.
const OPS_ROLES: UserRole[] = ["admin", "support", "catalog", "finance"];

/** Restricts a route subtree to operations roles. */
export function AdminRoute() {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return <Skeleton className="h-64 w-full" />;
  }
  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }
  if (!OPS_ROLES.includes(user.role)) {
    return <Navigate to="/login" replace />;
  }
  return <Outlet />;
}
