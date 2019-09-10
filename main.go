package main

import (
	"fmt"
	"os"
)

func main() {
	err := Run(os.Args[1:])
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}

// Run is the main loop, but returns an error
func Run(args []string) error {
	// TODO: remove this
	fmt.Println(args)
	cfg, err := GetConfigFromEnv()
	if err != nil {
		return err
	}
	bin := cfg.CurrentBin()
	err = EnsureBinary(bin)
	if err != nil {
		return err
	}
	return LaunchProcess(cfg, bin, args)
}
