package main

import (
	"os"

	"github.com/mothership/rds-auth-proxy/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
