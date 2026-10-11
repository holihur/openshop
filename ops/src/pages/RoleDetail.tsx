import { useEffect, useMemo, useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Save } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Skeleton } from "@lib/components/ui/skeleton";
import { errorMessage } from "@lib/errors";
import { useI18n } from "@lib/i18n";
import { useAuth } from "@lib/auth";
import {
  roleId,
  useCreateRole,
  useRoleMatrix,
  useUpdateRole,
  type RoleInput,
} from "@lib/hooks/useAdmin";

/** Groups the permission catalogue the way the console's menu is organised. */
function groupPermissions(permissions: string[]): [string, string[]][] {
  const groups = new Map<string, string[]>();
  for (const permission of permissions) {
    const area = permission.split(":")[0] ?? permission;
    groups.set(area, [...(groups.get(area) ?? []), permission]);
  }
  return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b));
}

/**
 * Create or edit one custom role. It is its own page rather than a panel beside
 * the list: choosing permissions is a considered task, and the list should stay
 * a list.
 */
export function RoleNewPage() {
  const { t } = useI18n();
  const { user } = useAuth();
  const navigate = useNavigate();
  const { key = "" } = useParams();
  const editing = key !== "";
  const { data, isLoading } = useRoleMatrix();
  const create = useCreateRole();
  const update = useUpdateRole();

  const [draft, setDraft] = useState<RoleInput>({ key: "", name: "", description: "", permissions: [] });

  const current = useMemo(
    () => (editing ? (data ?? []).find((role) => roleId(role) === key) : undefined),
    [data, editing, key],
  );
  const catalog = data?.[0]?.catalog ?? [];
  const groups = useMemo(() => groupPermissions(catalog), [catalog]);
  // A role can only grant what its author holds, so the picker hides the rest.
  const grantable = useMemo(() => new Set(user?.permissions ?? catalog), [user, catalog]);

  useEffect(() => {
    if (current) {
      setDraft({
        key: roleId(current),
        name: current.name ?? roleId(current),
        description: current.description,
        permissions: [...current.permissions],
      });
    }
  }, [current]);

  function toggle(permission: string) {
    setDraft((current) => ({
      ...current,
      permissions: current.permissions.includes(permission)
        ? current.permissions.filter((p) => p !== permission)
        : [...current.permissions, permission],
    }));
  }

  function onSubmit(event: FormEvent) {
    event.preventDefault();
    const done = () => {
      toast.success(editing ? t("roles.saved") : t("roles.created"));
      navigate("/roles");
    };
    const failed = (error: Error) => toast.error(errorMessage(error));
    if (editing) {
      update.mutate(
        { key, input: { name: draft.name, description: draft.description, permissions: draft.permissions } },
        { onSuccess: done, onError: failed },
      );
      return;
    }
    create.mutate(draft, { onSuccess: done, onError: failed });
  }

  if (editing && isLoading) return <Skeleton className="h-64 w-full" />;
  if (editing && !isLoading && !current) {
    return <p className="text-muted-foreground">{t("roles.notFound")}</p>;
  }
  if (editing && current && !current.custom) {
    // A built-in role's meaning lives in code, so it cannot be edited here.
    return <p className="text-muted-foreground">{t("roles.builtinReadOnly")}</p>;
  }

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <div>
        <Link to="/roles" className="text-muted-foreground inline-flex items-center gap-1 text-sm">
          <ArrowLeft className="size-4" />
          {t("roles.title")}
        </Link>
        <h1 className="text-xl font-semibold">{editing ? t("roles.edit") : t("roles.new")}</h1>
        <p className="text-muted-foreground text-sm">{t("roles.ceilingNote")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{editing ? draft.key : t("roles.new")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1">
                <Label htmlFor="r-key">{t("roles.key")}</Label>
                <Input
                  id="r-key"
                  value={draft.key ?? ""}
                  disabled={editing}
                  required
                  pattern="[a-z][a-z0-9_-]{1,63}"
                  onChange={(e) => setDraft({ ...draft, key: e.target.value.toLowerCase() })}
                />
                <p className="text-muted-foreground text-xs">{t("roles.keyHint")}</p>
              </div>
              <div className="space-y-1">
                <Label htmlFor="r-name">{t("roles.name")}</Label>
                <Input
                  id="r-name"
                  value={draft.name}
                  required
                  onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                />
              </div>
            </div>
            <div className="space-y-1">
              <Label htmlFor="r-desc">{t("roles.description")}</Label>
              <Input
                id="r-desc"
                value={draft.description ?? ""}
                onChange={(e) => setDraft({ ...draft, description: e.target.value })}
              />
            </div>

            <div className="space-y-2">
              <Label>{t("roles.permissions")}</Label>
              <p className="text-muted-foreground text-xs">
                {t("roles.selected", { count: draft.permissions.length })}
              </p>
              <div className="grid gap-2 sm:grid-cols-2">
                {groups.map(([area, permissions]) => (
                  <fieldset key={area} className="rounded-md border p-2">
                    <legend className="text-muted-foreground px-1 text-xs font-medium">{area}</legend>
                    <div className="flex flex-col gap-1">
                      {permissions.map((permission) => (
                        <label
                          key={permission}
                          className="flex items-center gap-1.5 font-mono text-xs"
                          title={grantable.has(permission) ? undefined : t("roles.notGrantable")}
                        >
                          <input
                            type="checkbox"
                            checked={draft.permissions.includes(permission)}
                            disabled={!grantable.has(permission)}
                            onChange={() => toggle(permission)}
                          />
                          {permission}
                        </label>
                      ))}
                    </div>
                  </fieldset>
                ))}
              </div>
            </div>

            <div className="flex gap-2">
              <Button type="submit" disabled={create.isPending || update.isPending}>
                <Save className="size-4" />
                {editing ? t("common.save") : t("roles.create")}
              </Button>
              <Button type="button" variant="outline" asChild>
                <Link to="/roles">{t("common.cancel")}</Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
