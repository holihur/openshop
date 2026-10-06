// Command migrate applies the versioned SQL migrations. It is safe to run on
// every replica at deploy time: a PostgreSQL advisory lock serialises runners,
// so only one applies each version.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/holihur/openshop/internal/adapter/postgres"
	"github.com/holihur/openshop/internal/config"
)

func main() {
	dir := flag.String("dir", "migrations", "directory containing .sql migrations")
	flag.Parse()

	_ = godotenv.Load(".env", "../.env")

	cfg, err := config.Load()
	if err != nil {
		fatal("config", err)
	}

	migrations, err := loadMigrations(*dir)
	if err != nil {
		fatal("load migrations", err)
	}
	if len(migrations) == 0 {
		fmt.Println("no migrations found in", *dir)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := postgres.Open(cfg.Postgres, nil)
	if err != nil {
		fatal("connect", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx, migrations); err != nil {
		fatal("migrate", err)
	}
	fmt.Printf("applied %d migration(s) successfully\n", len(migrations))
}

func loadMigrations(dir string) ([]postgres.Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	out := make([]postgres.Migration, 0, len(names))
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		out = append(out, postgres.Migration{
			Version: strings.TrimSuffix(name, ".sql"),
			SQL:     string(body),
		})
	}
	return out, nil
}

func fatal(step string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", step, err)
	os.Exit(1)
}
