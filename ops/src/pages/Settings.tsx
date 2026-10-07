import { useEffect, useMemo, useState, type FormEvent } from "react";
import { NavLink, useParams } from "react-router-dom";
import { RotateCcw, Save } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Skeleton } from "@lib/components/ui/skeleton";
import { useI18n } from "@lib/i18n";
import type { MessageKey } from "@lib/i18n/messages";
import { cn } from "@lib/utils";
import { useSettings, useUpdateSettings } from "@lib/hooks/useAdmin";
import type { Setting } from "@lib/types";

const GROUP_ORDER = ["store", "checkout", "inventory", "auth", "security", "payment", "mail"];

/** Runtime configuration: a group sub-menu on the left, the group's settings on
 * the right. Everything here applies without a restart. */
export function SettingsPage() {
  const { t } = useI18n();
  const { data, isLoading } = useSettings();
  const save = useUpdateSettings();
  const [draft, setDraft] = useState<Record<string, string>>({});
  const { group: groupParam } = useParams();

  useEffect(() => {
    if (!data) return;
    const next: Record<string, string> = {};
    for (const s of data) next[s.key] = s.value;
    setDraft(next);
  }, [data]);

  const groups = useMemo(() => {
    const by = new Map<string, Setting[]>();
    for (const s of data ?? []) {
      const list = by.get(s.group) ?? [];
      list.push(s);
      by.set(s.group, list);
    }
    return [...by.entries()].sort(
      (a, b) => GROUP_ORDER.indexOf(a[0]) - GROUP_ORDER.indexOf(b[0]),
    );
  }, [data]);

  // The active group is a route param, so each section is deep-linkable.
  const activeGroup = groupParam ?? groups[0]?.[0] ?? "";

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  const changed = (data ?? []).filter((s) => draft[s.key] !== undefined && draft[s.key] !== s.value);
  const groupChanged = (items: Setting[]) =>
    items.some((s) => draft[s.key] !== undefined && draft[s.key] !== s.value);
  const current = groups.find(([group]) => group === activeGroup);

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (changed.length === 0) return;
    const payload: Record<string, string> = {};
    for (const s of changed) payload[s.key] = draft[s.key];
    save.mutate(payload);
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-muted-foreground text-sm">{t("ops.settingsHint")}</p>
        <Button type="submit" size="sm" disabled={changed.length === 0 || save.isPending}>
          <Save className="size-4" />
          {t("common.save")}
        </Button>
      </div>

      <div className="flex flex-col gap-4 md:flex-row md:gap-6">
        <nav className="md:w-48 md:shrink-0">
          <ul className="flex gap-1 overflow-x-auto pb-1 md:flex-col md:overflow-visible md:pb-0">
            {groups.map(([group, items]) => (
              <li key={group}>
                <NavLink
                  to={`/settings/${group}`}
                  className={cn(
                    "flex w-full items-center justify-between gap-2 whitespace-nowrap rounded-md px-3 py-2 text-left text-sm font-medium transition-colors",
                    activeGroup === group
                      ? "bg-accent text-accent-foreground"
                      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
                  )}
                >
                  {t(`settings.group.${group}` as MessageKey)}
                  {groupChanged(items) && (
                    <span className="bg-primary size-2 rounded-full" aria-hidden />
                  )}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>

        <div className="min-w-0 flex-1 space-y-4">
          {current && (
            <Card key={current[0]}>
              <CardHeader>
                <CardTitle>{t(`settings.group.${current[0]}` as MessageKey)}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {current[1].map((s) => (
                  <div key={s.key} className="grid gap-2 sm:grid-cols-[260px_1fr] sm:items-start">
                    <div>
                      <Label htmlFor={s.key}>{t(`settings.key.${s.key}` as MessageKey)}</Label>
                      <p className="text-muted-foreground text-xs">
                        {t(`settings.desc.${s.key}` as MessageKey)}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      {s.type === "bool" ? (
                        <input
                          id={s.key}
                          type="checkbox"
                          className="size-4"
                          checked={draft[s.key] === "true"}
                          onChange={(e) =>
                            setDraft((d) => ({ ...d, [s.key]: String(e.target.checked) }))
                          }
                        />
                      ) : s.type === "color" ? (
                        <div className="flex items-center gap-2">
                          <input
                            id={s.key}
                            type="color"
                            aria-label={t(`settings.key.${s.key}` as MessageKey)}
                            value={draft[s.key] || "#000000"}
                            onChange={(e) =>
                              setDraft((d) => ({ ...d, [s.key]: e.target.value }))
                            }
                            className="border-input h-9 w-12 cursor-pointer rounded-md border bg-transparent"
                          />
                          <Input
                            value={draft[s.key] ?? ""}
                            onChange={(e) => setDraft((d) => ({ ...d, [s.key]: e.target.value }))}
                            placeholder="#6366f1"
                            className="max-w-40"
                          />
                        </div>
                      ) : s.type === "json" ? (
                        <textarea
                          id={s.key}
                          value={draft[s.key] ?? ""}
                          onChange={(e) => setDraft((d) => ({ ...d, [s.key]: e.target.value }))}
                          spellCheck={false}
                          className="border-input bg-background min-h-32 w-full rounded-md border p-2 font-mono text-xs"
                        />
                      ) : (
                        <Input
                          id={s.key}
                          type={s.type === "int" ? "number" : "text"}
                          min={s.min}
                          max={s.max}
                          value={draft[s.key] ?? ""}
                          onChange={(e) => setDraft((d) => ({ ...d, [s.key]: e.target.value }))}
                        />
                      )}
                      {s.value !== s.default && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => setDraft((d) => ({ ...d, [s.key]: s.default }))}
                        >
                          <RotateCcw className="size-4" />
                          {t("ops.resetDefault")}
                        </Button>
                      )}
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </form>
  );
}
