package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"srv.exe.dev/srv"
)

//go:embed dist
var frontendDist embed.FS

var flagListenAddr = flag.String("listen", ":8000", "address to listen on")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func run() error {
	flag.Parse()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	server, err := srv.New("db.sqlite3", hostname)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	
	// Get frontend dist
	frontendFS, err := fs.Sub(frontendDist, "dist")
	if err != nil {
		return fmt.Errorf("frontend fs: %w", err)
	}
	
	return server.Serve(*flagListenAddr, frontendFS)
}
// rebuild trigger


