package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func main() {
	var err, usageHelp string
	var command string
	var commandFn appCommand
	var commandDescr appCommandDescriptor
	var commandArgs []string
	argsLen := len(os.Args)
	if argsLen < 2 {
		err = "missing the required <command> argument"
	} else {
		command = os.Args[1]
		for cmd, cmdDC := range getSupportedCommands() {
			cmdFn := cmdDC.V1
			cmdDescr := cmdDC.V2
			if cmd == command {
				commandFn = cmdFn
				commandDescr = cmdDescr
				minArgCnt := 2 + cmdDescr.reqArgCnt
				maxArgCnt := minArgCnt + cmdDescr.optArgCnt
				if argsLen < minArgCnt || argsLen > maxArgCnt {
					err = "invalid " + command + " command usage"
				}
				if err != "" {
					usageHelp = "usage:\n\n" + cmdDescr.usage
				} else {
					commandArgs = os.Args[2:]
				}
				break
			}
		}
		if commandFn == nil {
			err = "unknown command: " + command
		}
	}
	if commandFn != nil && err == "" && usageHelp == "" {
		var config appConfig
		if commandDescr.reqConfig {
			config = readConfig()
		}
		initializeAndRunCommand(commandFn, commandDescr, config, commandArgs)
	} else {
		if err != "" {
			fmt.Println("")
			fmt.Println(err)
		}
		usage(usageHelp)
	}
}

func initializeAndRunCommand(cmdFn appCommand, cmdDescr appCommandDescriptor, config appConfig, commandArgs []string) {
	cmdFn(config, commandArgs...)
	if cmdDescr.command != commandVersion.command && cmdDescr.command != commandHelp.command {
		fmt.Println("")
		fmt.Println("[ ------- done ------- ]")
	}
}

func usage(usageHelp string) {
	if usageHelp != "" {
		fmt.Println("")
		fmt.Println(usageHelp)
	} else {
		fmt.Println("")
		fmt.Println("usage:")
		fmt.Println("")
		fmt.Println("mbgen <command> [options]")
		scs := getSupportedCommands()
		var commands []string
		for _, sc := range scs {
			cmd := sc.V2.command
			commands = append(commands, cmd)
		}
		sort.Strings(commands)
		fmt.Println("")
		fmt.Println("where <command> is one of the following: " + strings.Join(commands, ", "))
		fmt.Println("")
		fmt.Println("use the following to get help on a specific command:")
		fmt.Println("")
		fmt.Println("mbgen help <command>")
		fmt.Println("")
	}
}
