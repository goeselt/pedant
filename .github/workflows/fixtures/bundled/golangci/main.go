// Package main is an intentionally malformed Go program for pedant e2e testing.
//
// Expected findings:
//
//	golangci-lint dupword  -- duplicate word in comment ("the the")
//	golangci-lint errcheck -- unchecked error returns (resp.Body.Close, openFile call, fetch call)
//	golangci-lint nilerr   -- err != nil branch returns nil instead of err
//	golangci-lint noctx    -- http.NewRequest must use http.NewRequestWithContext
package main

import (
	"context"
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

// fetch makes an HTTP request without using the provided context.
func fetch(ctx context.Context) error {
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func main() {
	openFile("/tmp/test")
	fetch(context.Background())
}
