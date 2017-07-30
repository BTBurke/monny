package main

import (
	"fmt"
	"os"

	"github.com/BTBurke/wtf/monitor"
)

func main() {

	monitor.ParseCommandLine()

	cmd, err := monitor.New(os.Args[1:], "test", monitor.JSONAlert("labels.responseCode", "404"))
	if err != nil {
		fmt.Println("Error in config: ", err)
		os.Exit(1)
	}

	if err := cmd.Exec(); err != nil {
		fmt.Println("Error while run: ", err)
		os.Exit(1)
	}

	os.Exit(0)
}
