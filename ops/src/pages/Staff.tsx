import { useMemo, useState, type FormEvent } from "react";
import { KeyRound, Plus, ShieldCheck, UserCog } from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Skeleton } from "@lib/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@lib/components/ui/table";
import { errorMessage } from "@lib/errors";
import { useI18n } from "@lib/i18n";
import { appropriateRoles } from "@lib/staff";
import { useAuth } from "@lib/auth";
import {
  useCreateStaff,
  useResetStaffPassword,
  useRoleMatrix,
  useStaff,
  useUpdateStaff,
} from "@lib/hooks/useAdmin";
import type { StaffMember } from "@lib/hooks/useAdmin";

/**
 * Staff and roles. This is the screen that answers "who can operate the
 * console, and what may they do": the roles tab shows the same permission
 * matrix the API enforces, and every change here is audited.
 */
export function StaffPage() {
  const { t } = useI18n();
  const { user } = useAuth();
  const { data, isLoading } = useStaff();
  const { data: roles } = useRoleMatrix();
  const create = useCreateStaff();
  const update = useUpdateStaff();
  const resetPassword = useResetStaffPassword();

  const [tab, setTab] = useState<"staff" | "roles">("staff");
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [role, setRole] = useState("support");
  const [issued, setIssued] = useState<{ email: string; password: string } | null>(null);

  const assignable = useMemo(() => appropriateRoles(), []);
  const members = data?.items ?? [];

  function onCreate(event: FormEvent) {
    event.preventDefault();
    create.mutate(
      { email, name: name || undefined, role },
      {
        onSuccess: (result) => {
          setIssued(
            result.generatedPassword
              ? { email: result.staff.email, password: result.generatedPassword }
              : null,
          );
          setEmail("");
          setName("");
          toast.success(t("staff.created"));
        },
        onError: (error: Error) => toast.error(errorMessage(error)),
      },
    );
  }

  function onReset(member: StaffMember) {
    resetPassword.mutate(member.id, {
      onSuccess: (result) =>
        setIssued(
          result.generatedPassword
            ? { email: result.staff.email, password: result.generatedPassword }
            : null,
        ),
      onError: (error: Error) => toast.error(errorMessage(error)),
    });
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">{t("staff.title")}</h1>
          <p className="text-muted-foreground text-sm">{t("staff.subtitle")}</p>
        </div>
        <div className="flex gap-1">
          <Button
            variant={tab === "staff" ? "default" : "outline"}
            size="sm"
            onClick={() => setTab("staff")}
          >
            <UserCog className="size-4" />
            {t("staff.tabStaff")}
          </Button>
          <Button
            variant={tab === "roles" ? "default" : "outline"}
            size="sm"
            onClick={() => setTab("roles")}
          >
            <ShieldCheck className="size-4" />
            {t("staff.tabRoles")}
          </Button>
        </div>
      </div>

      {/* A generated password is shown exactly once: it is never retrievable. */}
      {issued && (
        <Card className="border-amber-500/40">
          <CardContent className="space-y-1 py-4">
            <p className="text-sm font-medium">{t("staff.passwordOnce")}</p>
            <p className="font-mono text-sm break-all">{issued.password}</p>
            <p className="text-muted-foreground text-xs">
              {t("staff.passwordFor", { email: issued.email })}
            </p>
            <Button variant="outline" size="sm" onClick={() => setIssued(null)}>
              {t("common.dismiss")}
            </Button>
          </CardContent>
        </Card>
      )}

      {tab === "staff" ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle>{t("staff.add")}</CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={onCreate} className="grid gap-3 sm:grid-cols-4">
                <div className="space-y-1 sm:col-span-2">
                  <Label htmlFor="s-email">{t("staff.email")}</Label>
                  <Input
                    id="s-email"
                    type="email"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="s-name">{t("staff.name")}</Label>
                  <Input id="s-name" value={name} onChange={(e) => setName(e.target.value)} />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="s-role">{t("staff.role")}</Label>
                  <select
                    id="s-role"
                    value={role}
                    onChange={(e) => setRole(e.target.value)}
                    className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
                  >
                    {assignable.map((r) => (
                      <option key={r} value={r}>
                        {t(`staff.role.${r}` as never)}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="sm:col-span-4">
                  <Button type="submit" size="sm" disabled={create.isPending}>
                    <Plus className="size-4" />
                    {t("staff.addButton")}
                  </Button>
                  <span className="text-muted-foreground ml-3 text-xs">
                    {t("staff.passwordHint")}
                  </span>
                </div>
              </form>
            </CardContent>
          </Card>

          <Card className="py-0">
            <CardContent className="px-0">
              {isLoading ? (
                <div className="p-4">
                  <Skeleton className="h-10 w-full" />
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("staff.name")}</TableHead>
                      <TableHead>{t("staff.role")}</TableHead>
                      <TableHead>{t("common.status")}</TableHead>
                      <TableHead>{t("staff.permissions")}</TableHead>
                      <TableHead className="text-right">{t("common.actions")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {members.map((member) => {
                      const self = member.id === user?.id;
                      return (
                        <TableRow key={member.id}>
                          <TableCell>
                            <p className="font-medium">{member.name}</p>
                            <p className="text-muted-foreground text-xs">{member.email}</p>
                          </TableCell>
                          <TableCell>
                            <select
                              value={member.role}
                              disabled={self || update.isPending}
                              aria-label={t("staff.role")}
                              onChange={(e) =>
                                update.mutate(
                                  { id: member.id, input: { role: e.target.value } },
                                  { onError: (error: Error) => toast.error(errorMessage(error)) },
                                )
                              }
                              className="border-input bg-background h-8 rounded-md border px-2 text-sm disabled:opacity-60"
                            >
                              {assignable.map((r) => (
                                <option key={r} value={r}>
                                  {t(`staff.role.${r}` as never)}
                                </option>
                              ))}
                            </select>
                          </TableCell>
                          <TableCell>
                            <Badge variant={member.status === "active" ? "success" : "secondary"}>
                              {member.status === "active" ? t("common.active") : t("common.inactive")}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-muted-foreground text-xs">
                            {member.permissions.length}
                          </TableCell>
                          <TableCell className="text-right">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() =>
                                update.mutate(
                                  {
                                    id: member.id,
                                    input: {
                                      status: member.status === "active" ? "disabled" : "active",
                                    },
                                  },
                                  { onError: (error: Error) => toast.error(errorMessage(error)) },
                                )
                              }
                              disabled={self || update.isPending}
                            >
                              {member.status === "active" ? t("ops.disable") : t("ops.enable")}
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => onReset(member)}
                              disabled={resetPassword.isPending}
                            >
                              <KeyRound className="size-4" />
                              {t("staff.resetPassword")}
                            </Button>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
          <p className="text-muted-foreground text-xs">{t("staff.guards")}</p>
        </>
      ) : (
        <div className="space-y-4">
          {roles?.map((matrix) => (
            <Card key={matrix.role}>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  {t(`staff.role.${matrix.role}` as never)}
                  <Badge variant="secondary">
                    {t("staff.members", { count: matrix.members })}
                  </Badge>
                </CardTitle>
                <p className="text-muted-foreground text-sm">{matrix.description}</p>
              </CardHeader>
              <CardContent className="flex flex-wrap gap-1">
                {matrix.role === "admin" ? (
                  <Badge variant="success">{t("staff.everything")}</Badge>
                ) : (
                  matrix.permissions.map((permission) => (
                    <Badge key={permission} variant="outline" className="font-mono text-xs">
                      {permission}
                    </Badge>
                  ))
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
