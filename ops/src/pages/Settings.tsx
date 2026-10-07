import { useEffect, useMemo, useState, type FormEvent } from "react";
import { RotateCcw, Save } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Skeleton } from "@lib/components/ui/skeleton";
import { useI18n } from "@lib/i18n";
import type { MessageKey } from "@lib/i18n/messages";
import { useSettings, useUpdateSettings } from "@lib/hooks/useAdmin";
import type { Setting } from "@lib/types";

const GROUP_ORDER = ["store", "checkout", "inventory", "auth", "security"];

/** Runtime configuration: everything here applies without a restart. */
export function SettingsPage() {
  const { t } = useI18n();
  const { data, isLoading } = useSettings();
  const save = useUpdateSettings();
  const [draft, setDraft] = useState<Record<string, string>>({});

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

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  const changed = (data ?? []).filter((s) => draft[s.key] !== undefined && draft[s.key] !== s.value);

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

      {groups.map(([group, items]) => (
        <Card key={group}>
          <CardHeader>
            <CardTitle>{t(`settings.group.${group}` as MessageKey)}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {items.map((s) => (
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
      ))}
    </form>
  );
}
