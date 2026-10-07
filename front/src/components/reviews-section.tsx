import { useState, type FormEvent } from "react";
import { Star } from "lucide-react";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Separator } from "@lib/components/ui/separator";
import { Textarea } from "@lib/components/ui/textarea";
import { Skeleton } from "@lib/components/ui/skeleton";
import { useAddReview, useReviews } from "@lib/hooks/useReviews";
import { useAuth } from "@lib/auth";
import { useI18n, t as translate } from "@lib/i18n";
import { formatDate } from "@lib/format";
import { cn } from "@lib/utils";

function Stars({ value, className }: { value: number; className?: string }) {
  return (
    <span className={cn("inline-flex", className)} aria-label={translate("reviews.ratingAria", { value })}>
      {[1, 2, 3, 4, 5].map((n) => (
        <Star
          key={n}
          className={cn(
            "size-4",
            n <= Math.round(value) ? "fill-amber-400 text-amber-400" : "text-muted-foreground",
          )}
        />
      ))}
    </span>
  );
}

export function ReviewsSection({ productId }: { productId: string }) {
  const { user } = useAuth();
  const { t } = useI18n();
  const { data, isLoading } = useReviews(productId);
  const addReview = useAddReview(productId);

  const [rating, setRating] = useState(5);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    addReview.mutate(
      { rating, title, body },
      {
        onSuccess: () => {
          setTitle("");
          setBody("");
          setRating(5);
        },
      },
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("reviews.title")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {user && (
          <form onSubmit={onSubmit} className="space-y-3">
            <div className="space-y-2">
              <Label>{t("reviews.yourRating")}</Label>
              <div className="flex gap-1">
                {[1, 2, 3, 4, 5].map((n) => (
                  <button
                    key={n}
                    type="button"
                    onClick={() => setRating(n)}
                    aria-label={t("reviews.starsAria", { count: n })}
                  >
                    <Star
                      className={cn(
                        "size-6",
                        n <= rating ? "fill-amber-400 text-amber-400" : "text-muted-foreground",
                      )}
                    />
                  </button>
                ))}
              </div>
            </div>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("reviews.summaryPlaceholder")}
            />
            <Textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder={t("reviews.bodyPlaceholder")}
            />
            <Button type="submit" disabled={addReview.isPending}>
              {addReview.isPending ? t("reviews.submitting") : t("product.submitReview")}
            </Button>
            <Separator />
          </form>
        )}

        {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : !data || data.items.length === 0 ? (
          <p className="text-muted-foreground text-sm">{t("product.noReviews")}</p>
        ) : (
          <ul className="space-y-4">
            {data.items.map((review) => (
              <li key={review.id} className="space-y-1">
                <div className="flex items-center gap-3">
                  <Stars value={review.rating} />
                  {review.title && <span className="font-medium">{review.title}</span>}
                  {review.verifiedPurchase && (
                    <Badge variant="secondary" className="text-xs">
                      {t("product.verifiedPurchase")}
                    </Badge>
                  )}
                  <span className="text-muted-foreground ml-auto text-xs">
                    {formatDate(review.createdAt)}
                  </span>
                </div>
                {review.body && <p className="text-muted-foreground text-sm">{review.body}</p>}
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
