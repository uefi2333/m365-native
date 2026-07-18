package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"m365-native/internal/mcp"
	"os"
	"time"
)

func main() {
	path := flag.String("config", "mcp.json", "MCP server config path")
	timeout := flag.Duration("timeout", 30*time.Second, "operation timeout")
	flag.Parse()
	cfg, err := mcp.LoadConfig(*path)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	manager := mcp.NewManager()
	defer manager.Close()
	if err := manager.Start(ctx, cfg); err != nil {
		log.Fatal(err)
	}
	b, err := json.MarshalIndent(manager.Tools(), "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	_, _ = os.Stdout.Write(append(b, '\n'))
	fmt.Printf("MCP probe succeeded: %d tool(s)\n", len(manager.Tools()))
}
