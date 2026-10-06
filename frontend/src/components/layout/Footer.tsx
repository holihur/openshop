export function Footer() {
  return (
    <footer className="text-muted-foreground mt-16 border-t">
      <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-2 px-4 py-8 text-sm sm:flex-row">
        <p>© {new Date().getFullYear()} OpenShop. Built with Go, Gin, GORM, PostgreSQL, NATS & Redis.</p>
        <p>Horizontally scalable by design.</p>
      </div>
    </footer>
  );
}
