package main

import "os"

func main() {
	_ = run()
}

func run() error {
	os.Exit(1) // want "calling os.Exit in main"
	return nil
}
