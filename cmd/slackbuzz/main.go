package main

import (
	"os"

	"github.com/isaacrowntree/slackbuzz-cli/internal/app"
)

func main() {
	code := app.Run()
	os.Exit(code)
}
