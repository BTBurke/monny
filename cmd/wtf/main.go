package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/BTBurke/wtf/monitor"
)

func main() {

	usercmd, opts := monitor.ParseCommandLine()
	fmt.Printf("Usercmd: %s\n", strings.Join(usercmd, " "))

	cmd, err := monitor.New(usercmd, "test", opts...)
	if err != nil {
		fmt.Println("Error in config:")
		for _, e := range err {
			fmt.Println(e)
		}
		os.Exit(1)
	}

	if err := cmd.Exec(); err != nil {
		fmt.Println("Error while run: ", err)
		os.Exit(1)
	}

	os.Exit(0)
}
