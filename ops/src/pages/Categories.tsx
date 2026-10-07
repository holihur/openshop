import { useState, type FormEvent } from "react";
import { Pencil, Plus } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
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
import { useCategories, useCreateCategory, useUpdateCategory } from "@lib/hooks/useAdmin";

/** Manage the product categories that populate the product form. */
export function CategoriesPage() {
  const { t } = useI18n();
  const { data: categories, isLoading } = useCategories();
  const create = useCreateCategory();
  const update = useUpdateCategory();

  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");

  function onCreate(e: FormEvent) {
    e.preventDefault();
    create.mutate(
      { name, slug: slug || undefined },
      {
        onSuccess: () => {
          setName("");
          setSlug("");
        },
      },
    );
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>{t("ops.newCategory")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onCreate} className="flex flex-wrap items-end gap-3">
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
            <Button type="submit" size="sm" disabled={create.isPending}>
              <Plus className="size-4" />
              {t("ops.createCategory")}
            </Button>
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
                  <TableHead>{t("ops.categoryName")}</TableHead>
                  <TableHead>{t("ops.slug")}</TableHead>
                  <TableHead className="text-right">{t("common.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {categories?.map((c) => (
                  <TableRow key={c.id}>
                    <TableCell className="font-medium">
                      {editingId === c.id ? (
                        <Input
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          className="h-8 max-w-xs"
                        />
                      ) : (
                        c.name
                      )}
                    </TableCell>
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
                                { id: c.id, input: { name: editName } },
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
