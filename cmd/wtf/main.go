package main

import (
	"fmt"
	"os"

	"github.com/BTBurke/wtf/monitor"
)

func main() {

	usercmd, opts, err := monitor.ParseCommandLine()
	if err != nil {
		fmt.Printf("Could not parse configuration: %s\n\nUse xray --help for options\n", err)
		os.Exit(1)
	}

	cmd, errs := monitor.New(usercmd, opts...)
	if len(errs) > 0 {
		fmt.Println("Error in config:")
		for _, e := range errs {
			fmt.Println(e)
		}
		os.Exit(1)
	}

	if err := cmd.Exec(); err != nil {
		fmt.Println("Process error:", err)
		os.Exit(1)
	}
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Not all reports sent: %s\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
