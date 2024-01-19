package version

import (
	"fmt"
)

var (
	Build  = "N/A" // Версия сборки.
	Date   = "N/A" // Дата сборки.
	Commit = "N/A" // Последний коммит.
)

// Print печатает в стандартный вывод информацию о сборке.
func Print() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n\n", Build, Date, Commit)
}
