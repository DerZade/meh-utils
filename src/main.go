package main

import (
	"flag"
	"fmt"
	"os"

	"./mvt"
	"./sat"
)

type command struct {
	name        string
	description string
	run         func(*flag.FlagSet)
}

var subCommands []command

func init() {
	subCommands = []command{
		command{"sat", "Build satellite tiles from grad_meh data.", sat.Run},
		command{"mvt", "Build mapbox vector tiles from grad_meh data.", mvt.Run},
		command{"help", "Print this message.", func(s *flag.FlagSet) { printUsage() }},
	}
}

func printUsage() {
	fmt.Printf("USAGE:\n    %s [SUBCOMMAND] [SUBCOMMAND FLAGS]\n\n", os.Args[0])
	fmt.Print("SUBCOMMANDS: \n")

	for i := 0; i < len(subCommands); i++ {
		name := subCommands[i].name

		fmt.Printf("%12s    %s\n", name, subCommands[i].description)
	}

	fmt.Printf("\nUse -h as SUBCOMMAND FLAG to print help for each subcommand.\n\n")
}

func main() {

	if len(os.Args) < 2 {
		fmt.Printf("\nERROR: No subcommand was provided.\n\n")
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	for i := 0; i < len(subCommands); i++ {
		if subCommands[i].name == cmd {
			set := flag.NewFlagSet(cmd, flag.ExitOnError)
			subCommands[i].run(set)
			return
		}
	}

	fmt.Printf("\nERROR: Subcommand '%s' was not found.\n\n", cmd)
	printUsage()
}
