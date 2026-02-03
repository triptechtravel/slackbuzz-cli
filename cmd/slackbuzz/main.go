package main

import (
	"os"

	"github.com/triptechtravel/slackbuzz-cli/internal/app"
)

func main() {
	code := app.Run()
	os.Exit(code)
}
