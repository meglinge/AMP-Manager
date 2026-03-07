package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"ampmanager/internal/database"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "migrate" {
		printUsage()
		os.Exit(1)
	}

	flags := flag.NewFlagSet("migrate", flag.ExitOnError)
	sourceType := flags.String("source-type", "sqlite", "源数据库类型: sqlite|postgres")
	sourceDSN := flags.String("source", "./data/data.db", "源数据库路径或 PostgreSQL URL")
	targetType := flags.String("target-type", "postgres", "目标数据库类型: sqlite|postgres")
	targetDSN := flags.String("target", "postgres://postgres:mysecretpassword@localhost:5432/ampmanager?sslmode=disable", "目标数据库路径或 PostgreSQL URL")
	clearTarget := flags.Bool("clear-target", true, "迁移前清空目标数据库中的业务表")
	withArchive := flags.Bool("with-archive", true, "同时迁移请求详情归档数据")
	flags.Parse(os.Args[2:])

	sourceOptions, err := buildDatabaseOptions(*sourceType, *sourceDSN)
	if err != nil {
		log.Fatal(err)
	}
	targetOptions, err := buildDatabaseOptions(*targetType, *targetDSN)
	if err != nil {
		log.Fatal(err)
	}

	err = database.MigrateBetweenDatabases(database.MigrationParams{
		ClearTarget: *clearTarget,
		OnProgress: func(progress database.MigrationProgress) {
			log.Printf("PROGRESS %d %s", progress.Progress, progress.Message)
		},
		Source:      sourceOptions,
		Target:      targetOptions,
		WithArchive: *withArchive,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("迁移完成: %s -> %s", sourceOptions.Type, targetOptions.Type)
}

func printUsage() {
	fmt.Println(`用法:
  go run ./cmd/dbtool migrate --source-type sqlite --source ./data/data.db --target-type postgres --target postgres://postgres:mysecretpassword@localhost:5432/ampmanager?sslmode=disable
  go run ./cmd/dbtool migrate --source-type postgres --source postgres://postgres:mysecretpassword@localhost:5432/ampmanager?sslmode=disable --target-type sqlite --target ./data/data.db`)
}

func buildDatabaseOptions(rawType, rawTarget string) (database.Options, error) {
	trimmedType := strings.TrimSpace(strings.ToLower(rawType))
	trimmedTarget := strings.TrimSpace(rawTarget)

	switch database.DBType(trimmedType) {
	case database.DBTypeSQLite:
		return database.Options{Type: database.DBTypeSQLite, SQLitePath: trimmedTarget}.Normalize()
	case database.DBTypePostgres:
		return database.Options{Type: database.DBTypePostgres, DatabaseURL: trimmedTarget}.Normalize()
	default:
		return database.Options{}, fmt.Errorf("不支持的数据库类型: %s", rawType)
	}
}
