import { useRef, useState } from "react";
import { Download, Upload } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { useI18n } from "@lib/i18n";
import { exportProductsCsv, useImportProducts, type ImportReport } from "@lib/hooks/useProductCsv";

/**
 * CSV import/export for the catalogue. An upload is validated first and its
 * effect shown, so a file is never applied blind.
 */
export function ProductCsv() {
  const { t } = useI18n();
  const fileInput = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<ImportReport | null>(null);
  const [result, setResult] = useState<ImportReport | null>(null);
  const [exporting, setExporting] = useState(false);
  const importCsv = useImportProducts();

  function onExport() {
    setExporting(true);
    exportProductsCsv().finally(() => setExporting(false));
  }

  function onPick(selected: File | null) {
    setFile(selected);
    setPreview(null);
    setResult(null);
    if (selected) importCsv.mutate({ file: selected, dryRun: true }, { onSuccess: setPreview });
  }

  function onConfirm() {
    if (!file) return;
    importCsv.mutate({ file, dryRun: false }, { onSuccess: (report) => {
      setResult(report);
      setPreview(null);
      setFile(null);
    } });
  }

  function reset() {
    setFile(null);
    setPreview(null);
    setResult(null);
    if (fileInput.current) fileInput.current.value = "";
  }

  const report = result ?? preview;

  return (
    <>
      <Button variant="outline" size="sm" onClick={onExport} disabled={exporting}>
        <Download className="size-4" />
        {t("ops.exportCsv")}
      </Button>
      <Button variant="outline" size="sm" onClick={() => fileInput.current?.click()}>
        <Upload className="size-4" />
        {t("ops.importCsv")}
      </Button>
      <input
        ref={fileInput}
        type="file"
        accept=".csv,text/csv"
        aria-label={t("ops.importCsv")}
        className="hidden"
        onChange={(e) => onPick(e.target.files?.[0] ?? null)}
      />

      {report && (
        <Card className="w-full">
          <CardHeader className="flex-row items-center justify-between gap-2">
            <CardTitle className="text-base">
              {report.dryRun ? t("ops.importPreview") : t("ops.importDone")}
            </CardTitle>
            <div className="flex gap-2">
              {report.dryRun ? (
                <Button size="sm" onClick={onConfirm} disabled={importCsv.isPending}>
                  {t("ops.importConfirm")}
                </Button>
              ) : null}
              <Button size="sm" variant="ghost" onClick={reset}>
                {t("common.close")}
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <p className="text-muted-foreground">
              {t("ops.importSummary", {
                rows: report.rowsRead,
                created: report.productsCreated,
                updated: report.productsUpdated,
                variantsCreated: report.variantsCreated,
                variantsUpdated: report.variantsUpdated,
              })}
            </p>
            {report.errors && report.errors.length > 0 && (
              <div className="rounded-md border border-destructive/40 p-2">
                <p className="text-destructive font-medium">
                  {t("ops.importErrors", { count: report.errors.length })}
                </p>
                <ul className="text-muted-foreground mt-1 space-y-0.5 text-xs">
                  {report.errors.slice(0, 20).map((e) => (
                    <li key={`${e.line}-${e.message}`}>
                      {t("ops.importLine", { line: e.line })}: {e.message}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </>
  );
}
