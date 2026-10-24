package main

  import (
        "database/sql"
        "fmt"
        "os"
        "path/filepath"
        "sort"
        "strings"
  )

  func runMigrations(db *sql.DB) error {
        migrationsDir := "database/migrations"

        files, err := os.ReadDir(migrationsDir)
        if err != nil {
                return fmt.Errorf("error reading migrations dir: %w", err)
        }

        var sqlFiles []string
        for _, f := range files {
                if strings.HasSuffix(f.Name(), ".sql") {
                        sqlFiles = append(sqlFiles, f.Name())
                }
        }
        sort.Strings(sqlFiles)

        for _, file := range sqlFiles {
                path := filepath.Join(migrationsDir, file)
                content, err := os.ReadFile(path)
                if err != nil {
                        return fmt.Errorf("error reading %s: %w", file, err)
                }

                _, err = db.Exec(string(content))
                if err != nil {
                        return fmt.Errorf("error executing %s: %w", file, err)
                }
                fmt.Printf("✓ %s\n", file)
        }
        return nil
  }
