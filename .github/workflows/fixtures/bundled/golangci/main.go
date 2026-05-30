// Package main is an intentionally malformed Go program for pedant e2e testing.
//
// Expected findings:
//
//	golangci-lint errcheck  -- error return value of f.Close() not checked
package main

import "os"

func main() {
	f, err := os.Open("/tmp/test")
	if err != nil {
		return
	}
	f.Close()
}
