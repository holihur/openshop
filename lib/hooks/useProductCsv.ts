import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import { errorMessage } from "@lib/errors";
import { t } from "@lib/i18n";

export interface ImportReport {
  rowsRead: number;
  productsCreated: number;
  productsUpdated: number;
  variantsCreated: number;
  variantsUpdated: number;
  errors?: { line: number; message: string }[];
  dryRun: boolean;
}

/** Download the whole catalogue as CSV. */
export async function exportProductsCsv(): Promise<void> {
  await api.download("/ops/products/export.csv", "products.csv");
}

/**
 * Upload a CSV. A dry run validates the file and reports what it would change,
 * so nothing is written until it is confirmed.
 */
export function useImportProducts() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ file, dryRun }: { file: File; dryRun: boolean }) =>
      api.upload<ImportReport>(
        `/ops/products/import?dry_run=${dryRun ? "true" : "false"}`,
        file,
      ),
    onSuccess: (report) => {
      if (report.dryRun) return; // the caller shows the preview
      toast.success(
        t("toast.importDone", {
          created: report.productsCreated,
          updated: report.productsUpdated,
        }),
      );
      void queryClient.invalidateQueries({ queryKey: ["products"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}
