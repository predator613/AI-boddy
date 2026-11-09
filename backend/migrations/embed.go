// Package migrations встраивает SQL-файлы в бинарник, чтобы контейнеру не нужны были внешние файлы.
package migrations

import "embed"

// FS содержит все *.sql из этой папки. Применяются по порядку имен: 001_, 002_, ...
//
//go:embed *.sql
var FS embed.FS
