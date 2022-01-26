package main

import (
	"os"

	"tensor/lib/cli"
)


func  main(){
	defer os.Exit(0)
	cli := cli.CommandLine{}
	cli.Run()
}