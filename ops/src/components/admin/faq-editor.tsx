import { useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Skeleton } from "@lib/components/ui/skeleton";
import { Textarea } from "@lib/components/ui/textarea";
import { useI18n } from "@lib/i18n";
import { useProductFAQs, useReplaceFAQs } from "@lib/hooks/useAdmin";

interface Row {
  question: string;
  answer: string;
}

/** Edit a product's FAQs (question/answer pairs, shown on the storefront). */
export function FaqEditor({ productId }: { productId: string }) {
  const { t } = useI18n();
  const { data, isLoading } = useProductFAQs(productId);
  const save = useReplaceFAQs(productId);
  const [rows, setRows] = useState<Row[]>([]);

  useEffect(() => {
    if (data) setRows(data.map((f) => ({ question: f.question, answer: f.answer })));
  }, [data]);

  if (isLoading) {
    return <Skeleton className="h-32 w-full" />;
  }

  const set = (index: number, patch: Partial<Row>) =>
    setRows((list) => list.map((row, i) => (i === index ? { ...row, ...patch } : row)));

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>{t("ops.faq")}</CardTitle>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => setRows((list) => [...list, { question: "", answer: "" }])}
        >
          <Plus className="size-4" />
          {t("ops.addFaq")}
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        {rows.length === 0 && <p className="text-muted-foreground text-sm">{t("ops.noFaqs")}</p>}
        {rows.map((row, index) => (
          <div key={index} className="space-y-2 rounded-md border p-3">
            <div className="flex items-end gap-2">
              <div className="flex-1 space-y-1">
                <Label htmlFor={`faq-q-${index}`}>{t("ops.question")}</Label>
                <Input
                  id={`faq-q-${index}`}
                  value={row.question}
                  onChange={(e) => set(index, { question: e.target.value })}
                />
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={t("common.delete")}
                onClick={() => setRows((list) => list.filter((_, i) => i !== index))}
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
            <div className="space-y-1">
              <Label htmlFor={`faq-a-${index}`}>{t("ops.answer")}</Label>
              <Textarea
                id={`faq-a-${index}`}
                value={row.answer}
                rows={2}
                onChange={(e) => set(index, { answer: e.target.value })}
              />
            </div>
          </div>
        ))}
        <Button
          type="button"
          size="sm"
          disabled={save.isPending}
          onClick={() => save.mutate(rows)}
        >
          {t("common.save")}
        </Button>
      </CardContent>
    </Card>
  );
}
