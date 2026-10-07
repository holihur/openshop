import { useState, type FormEvent } from "react";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";
import { formatDate } from "@lib/format";
import {
  useCreateToken,
  useRevokeToken,
  useTokenScopes,
  useTokens,
  type TokenRealm,
} from "@lib/hooks/useTokens";
import type { ScopeInfo } from "@lib/types";

/** Create, list and revoke personal access tokens (storefront or console). */
export function TokenManager({ realm }: { realm: TokenRealm }) {
  const { t } = useI18n();
  const { user } = useAuth();
  const { data: scopes } = useTokenScopes(realm);
  const { data: tokens } = useTokens(realm);
  const create = useCreateToken(realm);
  const revoke = useRevokeToken(realm);

  const [name, setName] = useState("");
  const [selected, setSelected] = useState<string[]>([]);
  const [cidrs, setCidrs] = useState("");
  const [days, setDays] = useState("");
  const [secret, setSecret] = useState<string | null>(null);

  const granted = user?.permissions ?? [];
  // An operations token can never exceed the owner's role.
  const available = (scopes ?? []).filter((s) => {
    if (realm === "front") return true;
    if (s.scope === "*") return user?.role === "admin";
    return granted.includes(s.scope);
  });

  const groups = available.reduce<Record<string, ScopeInfo[]>>((acc, s) => {
    (acc[s.group] ??= []).push(s);
    return acc;
  }, {});

  function toggle(scope: string) {
    setSelected((cur) => (cur.includes(scope) ? cur.filter((s) => s !== scope) : [...cur, scope]));
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    create.mutate(
      {
        name,
        scopes: selected,
        cidrs: cidrs.split(/[\s,]+/).filter(Boolean),
        ...(Number(days) > 0 ? { expiresInDays: Number(days) } : {}),
      },
      {
        onSuccess: (created) => {
          setSecret(created.token);
          setName("");
          setSelected([]);
          setCidrs("");
          setDays("");
        },
      },
    );
  }

  return (
    <div className="space-y-6">
      {secret && (
        <Card className="border-emerald-500">
          <CardHeader>
            <CardTitle className="text-base">{t("tokens.created")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <p className="text-muted-foreground text-xs">{t("tokens.createdHint")}</p>
            <code className="bg-muted block overflow-x-auto rounded p-2 font-mono text-xs">
              {secret}
            </code>
            <Button variant="outline" size="sm" onClick={() => setSecret(null)}>
              {t("common.close")}
            </Button>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("tokens.new")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="space-y-2">
                <Label htmlFor="token-name">{t("tokens.name")}</Label>
                <Input
                  id="token-name"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder={t("tokens.namePlaceholder")}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="token-cidrs">{t("tokens.cidrs")}</Label>
                <Input
                  id="token-cidrs"
                  value={cidrs}
                  onChange={(e) => setCidrs(e.target.value)}
                  placeholder="10.0.0.0/8, 203.0.113.7"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="token-days">{t("tokens.expiresInDays")}</Label>
                <Input
                  id="token-days"
                  type="number"
                  min="0"
                  value={days}
                  onChange={(e) => setDays(e.target.value)}
                  placeholder="0"
                />
              </div>
            </div>
            <p className="text-muted-foreground text-xs">{t("tokens.cidrHint")}</p>

            <div className="space-y-3">
              <Label>{t("tokens.scopes")}</Label>
              {Object.entries(groups).map(([group, items]) => (
                <div key={group} className="space-y-1">
                  <p className="text-muted-foreground text-xs font-medium uppercase">{group}</p>
                  <div className="grid gap-1 sm:grid-cols-2">
                    {items.map((s) => (
                      <label key={s.scope} className="flex items-start gap-2 text-sm">
                        <input
                          type="checkbox"
                          className="mt-0.5"
                          checked={selected.includes(s.scope)}
                          onChange={() => toggle(s.scope)}
                        />
                        <span>
                          <span className="font-mono text-xs">{s.scope}</span>
                          <span className="text-muted-foreground block text-xs">
                            {s.description}
                          </span>
                        </span>
                      </label>
                    ))}
                  </div>
                </div>
              ))}
            </div>

            <Button type="submit" disabled={create.isPending || selected.length === 0}>
              {t("tokens.create")}
            </Button>
          </form>
        </CardContent>
      </Card>

      <div className="space-y-2">
        <h2 className="text-lg font-medium">{t("tokens.existing")}</h2>
        {tokens && tokens.length > 0 ? (
          <ul className="space-y-2">
            {tokens.map((token) => (
              <li
                key={token.id}
                className="flex flex-wrap items-center justify-between gap-2 rounded-md border p-3"
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{token.name}</span>
                    <code className="text-muted-foreground font-mono text-xs">
                      {token.prefix}…
                    </code>
                    {token.revokedAt && <Badge variant="secondary">{t("tokens.revoked")}</Badge>}
                  </div>
                  <p className="text-muted-foreground text-xs">
                    {token.scopes.join(", ")}
                    {token.cidrs.length > 0 && ` · ${token.cidrs.join(", ")}`}
                  </p>
                  <p className="text-muted-foreground text-xs">
                    {t("tokens.createdAt", { date: formatDate(token.createdAt) })}
                    {token.lastUsedAt && ` · ${t("tokens.lastUsed", { date: formatDate(token.lastUsedAt) })}`}
                    {token.expiresAt && ` · ${t("tokens.expiresAt", { date: formatDate(token.expiresAt) })}`}
                  </p>
                </div>
                {!token.revokedAt && (
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={revoke.isPending}
                    onClick={() => revoke.mutate(token.id)}
                  >
                    {t("tokens.revoke")}
                  </Button>
                )}
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-muted-foreground text-sm">{t("tokens.empty")}</p>
        )}
      </div>
    </div>
  );
}
