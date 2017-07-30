package main

import (
	"fmt"
	"os"

	"github.com/BTBurke/wtf/monitor"
)

func main() {

	usercmd, opts := monitor.ParseCommandLine()

	cmd, err := monitor.New(usercmd, opts...)
	if len(err) > 0 {
		fmt.Println("Error in config:")
		for _, e := range err {
			fmt.Println(e)
		}
		os.Exit(1)
	}

	if err := cmd.Exec(); err != nil {
		fmt.Println("Process error:", err)
		os.Exit(1)
	}

	os.Exit(0)
}
