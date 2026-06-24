// Package main is an intentionally malformed Go program for pedant e2e testing.
//
// Expected findings:
//
//	golangci-lint dupword  -- duplicate word in comment ("the the")
//	golangci-lint errcheck -- unchecked error returns (openFile call, f.Close, http.Get)
//	golangci-lint nilerr   -- err != nil branch returns nil instead of err
//	golangci-lint noctx    -- http.Get does not accept a context
package main

import (
	"net/http"
	"os"
)

// openFile opens the the file at path.
func openFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	f.Close()
	return nil
}

func main() {
	openFile("/tmp/test")
	http.Get("http://example.com")
}
