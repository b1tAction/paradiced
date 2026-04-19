// Package main provides the entry point for pdcli.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/b1tAction/paradiced/internal/cli/command"
)

func main() {
	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n正在退出...")
	}()

	// Execute CLI
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误：%v\n", err)
		os.Exit(1)
	}
}
