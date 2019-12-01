package main

import (
	"fmt"
	"os"
	"errors"

	"github.com/BTBurke/monny"
	"github.com/spf13/pflag"
)

func main() {

	usercmd, opts, err := monny.ParseCommandLine()
	if err != nil {
		if !errors.Is(err, pflag.ErrHelp) {
			fmt.Printf("Could not parse configuration: %s\n\nUse monny --help for options\n", err)
		}
		os.Exit(1)
	}

	cmd, errs := monny.New(usercmd, opts...)
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
