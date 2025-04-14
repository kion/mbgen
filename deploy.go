package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func rsyncDeploy(destination string) {
	if !dirExists(deployDirName) {
		sprintln("deploy dir does not exist - make sure to run the `mbgen generate` command first")
	} else {
		source, err := filepath.Abs(deployDirName)
		check(err)

		sprintln(
			" - deploy source: "+source,
			" - deploy destination: "+destination,
		)

		destPathSeparator := '/'
		for i := 0; i < len(destination); i++ {
			if destination[i] == '\\' {
				destPathSeparator = '\\'
				break
			}
		}

		deployOpts := []deployOptions{
			{
				source:      fmt.Sprintf("%s%c%s%c", source, os.PathSeparator, mediaDirName, os.PathSeparator),
				destination: fmt.Sprintf("%s%c%s", destination, destPathSeparator, mediaDirName),
			},
			{
				source:      fmt.Sprintf("%s%c%s%c", source, os.PathSeparator, deployPageDirName, os.PathSeparator),
				destination: fmt.Sprintf("%s%c%s", destination, destPathSeparator, deployPageDirName),
			},
			{
				source:      fmt.Sprintf("%s%c%s%c", source, os.PathSeparator, deployPostDirName, os.PathSeparator),
				destination: fmt.Sprintf("%s%c%s", destination, destPathSeparator, deployPostDirName),
			},
			{
				source:      fmt.Sprintf("%s%c%s%c", source, os.PathSeparator, deployPostsDirName, os.PathSeparator),
				destination: fmt.Sprintf("%s%c%s", destination, destPathSeparator, deployPostsDirName),
			},
			{
				source:      fmt.Sprintf("%s%c", source, os.PathSeparator),
				destination: destination,
				exclude: []string{
					mediaDirName,
					deployPageDirName,
					deployPostDirName,
					deployPostsDirName,
				},
			},
		}

		for i := 0; i < len(deployOpts); i++ {
			dOpts := deployOpts[i]
			fmt.Printf("\n - deploy: %s -> %s\n", dOpts.source, dOpts.destination)
			args := []string{
				"--archive",
				"--compress",
				"--delete",
				"--no-t",
				"--no-o",
				"--no-g",
				"--no-p",
				"--progress",
				"--verbose",
			}
			if len(dOpts.exclude) > 0 {
				for _, exclude := range dOpts.exclude {
					args = append(args, fmt.Sprintf("--exclude=%s", exclude))
				}
			}
			args = append(args, dOpts.source)
			args = append(args, dOpts.destination)
			cmd := exec.Command("rsync", args...)
			output, err := cmd.Output()
			check(err)
			outputLines := strings.Split(string(output), "\n")
			for _, line := range outputLines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "sent") ||
					strings.HasPrefix(line, "total") {
					fmt.Println(" - " + line)
				}
			}
			fmt.Printf(" - deploy: %s -> %s [complete]\n", dOpts.source, dOpts.destination)
		}
	}
}
