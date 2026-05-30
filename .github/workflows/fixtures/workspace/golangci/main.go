// Package main is a Go program for the golangci custom-config e2e scenario.
//
// Expected findings with custom config (godot enabled):
//
//	golangci-lint godot  -- function comment does not end with a period
//
// No findings with the bundled config (godot is not enabled there).
package main

// greet prints a greeting without a trailing period on this comment
func greet(name string) {
	println("Hello, " + name)
}

func main() {
	greet("World")
}
