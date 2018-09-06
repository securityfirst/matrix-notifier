package main

import (
	_ "github.com/lib/pq"
	"github.com/securityfirst/matrix-notifier/cmd"
)

func main() {
	cmd.Execute()
}
