import { useState, type FormEvent } from "react";
import { Plus } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { useI18n } from "@lib/i18n";
import { localeNames, supportedLocales } from "@lib/i18n/messages";
import { useCreateCategory } from "@lib/hooks/useAdmin";

/** Category creation form, used by its own page. */
export function CategoryForm({ onDone }: { onDone?: () => void }) {
  const { t } = useI18n();
  const create = useCreateCategory();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [names, setNames] = useState<Record<string, string>>({});

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    create.mutate(
      { name, slug: slug || undefined, names },
      { onSuccess: () => onDone?.() },
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("ops.newCategory")}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="space-y-3">
          <div className="flex flex-wrap items-end gap-3">
            <div className="space-y-1">
              <Label htmlFor="cat-name">{t("ops.categoryName")}</Label>
              <Input id="cat-name" value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div className="space-y-1">
              <Label htmlFor="cat-slug">{t("ops.slug")}</Label>
              <Input
                id="cat-slug"
                value={slug}
                onChange={(e) => setSlug(e.target.value)}
                placeholder="auto"
              />
            </div>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            {supportedLocales.map((locale) => (
              <div key={locale} className="space-y-1">
                <Label htmlFor={`cat-name-${locale}`}>{localeNames[locale]}</Label>
                <Input
                  id={`cat-name-${locale}`}
                  value={names[locale] ?? ""}
                  onChange={(e) => setNames((n) => ({ ...n, [locale]: e.target.value }))}
                  placeholder={name}
                />
              </div>
            ))}
          </div>
          <Button type="submit" size="sm" disabled={create.isPending}>
            <Plus className="size-4" />
            {t("ops.createCategory")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
