import { useState } from "react";
import { Link } from "react-router-dom";
import { Pencil, Plus } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Skeleton } from "@lib/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@lib/components/ui/table";
import { useI18n } from "@lib/i18n";
import { localeNames, supportedLocales } from "@lib/i18n/messages";
import { useCategories, useUpdateCategory } from "@lib/hooks/useAdmin";
import type { Category } from "@lib/types";

/** Manage the product categories and their per-locale translations. */
export function CategoriesPage() {
  const { t } = useI18n();
  const { data: categories, isLoading } = useCategories();
  const update = useUpdateCategory();

  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editNames, setEditNames] = useState<Record<string, string>>({});

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button size="sm" asChild>
          <Link to="/categories/new">
            <Plus className="size-4" />
            {t("ops.newCategory")}
          </Link>
        </Button>
      </div>

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
                  <TableHead>{t("ops.categoryName")}</TableHead>
                  {supportedLocales.map((locale) => (
                    <TableHead key={locale}>{localeNames[locale]}</TableHead>
                  ))}
                  <TableHead>{t("ops.slug")}</TableHead>
                  <TableHead className="text-right">{t("common.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {categories?.map((c: Category) => (
                  <TableRow key={c.id}>
                    <TableCell className="font-medium">
                      {editingId === c.id ? (
                        <Input
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          className="h-8 w-40"
                        />
                      ) : (
                        c.name
                      )}
                    </TableCell>
                    {supportedLocales.map((locale) => (
                      <TableCell key={locale} className="text-muted-foreground">
                        {editingId === c.id ? (
                          <Input
                            value={editNames[locale] ?? ""}
                            onChange={(e) =>
                              setEditNames((n) => ({ ...n, [locale]: e.target.value }))
                            }
                            placeholder={c.name}
                            className="h-8 w-40"
                          />
                        ) : (
                          c.names?.[locale] || "—"
                        )}
                      </TableCell>
                    ))}
                    <TableCell className="text-muted-foreground font-mono text-xs">{c.slug}</TableCell>
                    <TableCell className="text-right">
                      {editingId === c.id ? (
                        <div className="flex justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={update.isPending}
                            onClick={() =>
                              update.mutate(
                                { id: c.id, input: { name: editName, names: editNames } },
                                { onSuccess: () => setEditingId(null) },
                              )
                            }
                          >
                            {t("common.save")}
                          </Button>
                          <Button variant="ghost" size="sm" onClick={() => setEditingId(null)}>
                            {t("common.cancel")}
                          </Button>
                        </div>
                      ) : (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setEditingId(c.id);
                            setEditName(c.name);
                            setEditNames(c.names ?? {});
                          }}
                        >
                          <Pencil className="size-4" />
                          {t("common.edit")}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
