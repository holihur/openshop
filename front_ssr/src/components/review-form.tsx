"use client";

import { useActionState } from "react";

import { submitReview, type ReviewState } from "@/app/products/[id]/actions";

const initialState: ReviewState = { ok: false };

/**
 * The review form is a thin client wrapper around a server action. Without
 * JavaScript the browser posts the same form to the server, so writing a review
 * keeps working — the state hook only adds the inline feedback.
 */
export function ReviewForm({
  productId,
  labels,
}: {
  productId: string;
  labels: {
    title: string;
    body: string;
    rating: string;
    submit: string;
    sending: string;
    failed: string;
    thanks: string;
  };
}) {
  const [state, action, pending] = useActionState(submitReview, initialState);

  return (
    <form action={action} className="space-y-2 rounded-lg border p-4">
      <input type="hidden" name="productId" value={productId} />
      <div className="space-y-1">
        <label htmlFor="rating" className="text-sm font-medium">
          {labels.rating}
        </label>
        <select
          id="rating"
          name="rating"
          defaultValue="5"
          className="border-input h-9 rounded-md border px-3"
        >
          {[5, 4, 3, 2, 1].map((value) => (
            <option key={value} value={value}>
              {"★".repeat(value)}
            </option>
          ))}
        </select>
      </div>
      <input
        name="title"
        placeholder={labels.title}
        aria-label={labels.title}
        required
        className="border-input h-9 w-full rounded-md border px-3"
      />
      <textarea
        name="body"
        placeholder={labels.body}
        aria-label={labels.body}
        rows={3}
        className="border-input w-full rounded-md border px-3 py-2"
      />
      <button
        type="submit"
        disabled={pending}
        className="bg-primary text-primary-foreground h-9 rounded-md px-4 disabled:opacity-50"
      >
        {pending ? labels.sending : labels.submit}
      </button>
      {state.ok ? <p className="text-sm text-green-600">{labels.thanks}</p> : null}
      {state.message && !state.ok ? (
        <p className="text-sm text-red-600">{state.message === "unauthenticated" ? labels.failed : state.message}</p>
      ) : null}
    </form>
  );
}
