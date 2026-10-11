import { Link } from "react-router-dom";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Skeleton } from "@lib/components/ui/skeleton";
import { errorMessage } from "@lib/errors";
import { useI18n } from "@lib/i18n";
import { roleId, useDeleteRole, useRoleMatrix } from "@lib/hooks/useAdmin";

/**
 * The roles list. Built-in roles are read-only (their meaning is part of the
 * code); custom ones are created and edited on their own pages, so this screen
 * stays a list.
 */
export function RolesPage() {
  const { t } = useI18n();
  const { data, isLoading } = useRoleMatrix();
  const remove = useDeleteRole();

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">{t("roles.title")}</h1>
          <p className="text-muted-foreground text-sm">{t("roles.subtitle")}</p>
        </div>
        <Button size="sm" asChild>
          <Link to="/roles/new">
            <Plus className="size-4" />
            {t("roles.new")}
          </Link>
        </Button>
      </div>

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <div className="space-y-3">
          {data?.map((role) => {
            const id = roleId(role);
            const builtin = !role.custom;
            return (
              <Card key={id}>
                <CardHeader className="gap-1">
                  <CardTitle className="flex flex-wrap items-center gap-2 text-base">
                    {role.name ?? id}
                    <span className="text-muted-foreground font-mono text-xs">{id}</span>
                    <Badge variant={builtin ? "secondary" : "outline"}>
                      {builtin ? t("roles.builtin") : t("roles.custom")}
                    </Badge>
                    <Badge variant="secondary">{t("staff.members", { count: role.members })}</Badge>
                  </CardTitle>
                  <p className="text-muted-foreground text-sm">{role.description}</p>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex flex-wrap gap-1">
                    {id === "admin" ? (
                      <Badge variant="success">{t("staff.everything")}</Badge>
                    ) : (
                      role.permissions.map((permission) => (
                        <Badge key={permission} variant="outline" className="font-mono text-xs">
                          {permission}
                        </Badge>
                      ))
                    )}
                  </div>
                  {!builtin && (
                    <div className="flex gap-2">
                      <Button variant="outline" size="sm" asChild>
                        <Link to={`/roles/${id}`}>
                          <Pencil className="size-4" />
                          {t("common.edit")}
                        </Link>
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={remove.isPending || role.members > 0}
                        title={role.members > 0 ? t("roles.hasMembers") : undefined}
                        onClick={() =>
                          remove.mutate(id, {
                            onSuccess: () => toast.success(t("roles.deleted")),
                            onError: (error: Error) => toast.error(errorMessage(error)),
                          })
                        }
                      >
                        <Trash2 className="size-4" />
                        {t("common.delete")}
                      </Button>
                    </div>
                  )}
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
