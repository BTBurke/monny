package main

import (
	"fmt"
	"os"

	"github.com/BTBurke/wtf/command"
	"github.com/BTBurke/wtf/config"
)

func main() {
	cfg, err := config.New("test", config.Alert("vscode"), config.Daemon())
	if err != nil {
		fmt.Println("Error in config: ", err)
		os.Exit(1)
	}
	cmd := command.New([]string{"cat", "Gopkg.toml | cat \"test: \""}, cfg)

	if err := cmd.Exec(); err != nil {
		fmt.Println("Error while run: ", err)
		os.Exit(1)
	}

	os.Exit(0)
}
