package main

import (
	"log"
	"os"

	"github.com/spf13/cobra/doc"

	"shrugged/internal/cli"
)

func main() {
	outDir := "./docs"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatal(err)
	}

	cmd := cli.Root()
	cmd.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(cmd, outDir); err != nil {
		log.Fatal(err)
	}
}
