package main

import (
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
			printConfig(config)
		}
		initializeAndRunCommand(commandFn, commandDescr, config, commandArgs)
	} else {
		if err != "" {
			sprintln(err)
		}
		usage(usageHelp, 1)
	}
}

func initializeAndRunCommand(cmdFn appCommand, cmdDescr appCommandDescriptor, config appConfig, commandArgs []string) {
	cmdFn(config, commandArgs...)
	if cmdDescr.command != commandVersion.command && cmdDescr.command != commandHelp.command {
		sprintln("[ ------- done ------- ]")
	}
}

func usage(usageHelp string, exitCode int) {
	if usageHelp != "" {
		sprintln(usageHelp)
	} else {
		sprintln(
			"usage:",
			"mbgen <command> [options]",
		)
		scs := getSupportedCommands()
		var commands []string
		for _, sc := range scs {
			cmd := sc.V2.command
			commands = append(commands, cmd)
		}
		sort.Strings(commands)
		sprintln(
			"where <command> is one of the following:",
			" - "+strings.Join(commands, "\n - ")+"\n",
			"use the following to get help on a specific command:\n",
			"mbgen help <command>\n",
		)
	}
	os.Exit(exitCode)
}
