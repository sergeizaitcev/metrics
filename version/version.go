package version

import (
	"fmt"
	"io"
	"os"
)

var (
	Build  = "N/A" // Версия сборки.
	Date   = "N/A" // Дата сборки.
	Commit = "N/A" // Последний коммит.
)

// Print печатает в стандартный вывод информацию о сборке.
func Print() {
	PrintTo(os.Stdout)
}

// PrintTo печатает в w справочную информацию о сборке.
func PrintTo(w io.Writer) {
	fmt.Fprintf(w, "Build version: %s\nBuild date: %s\nBuild commit: %s\n", Build, Date, Commit)
}
