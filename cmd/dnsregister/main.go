package main

import (
	"os"

	// Frameworks
	"github.com/djthorpe/gopi/v3/pkg/tool"
)

func main() {
	os.Exit(tool.CommandLine("dnsregister", os.Args[1:], new(app)))
}
