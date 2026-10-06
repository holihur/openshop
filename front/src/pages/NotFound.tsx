import { Link } from "react-router-dom";

import { Button } from "@lib/components/ui/button";

export function NotFoundPage() {
  return (
    <div className="py-24 text-center">
      <p className="text-muted-foreground text-sm font-medium">404</p>
      <h1 className="mt-2 text-3xl font-bold">Page not found</h1>
      <p className="text-muted-foreground mt-2">
        The page you are looking for does not exist.
      </p>
      <Button className="mt-6" asChild>
        <Link to="/">Back to home</Link>
      </Button>
    </div>
  );
}
