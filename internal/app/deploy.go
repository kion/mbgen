package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// buildDeployOptions returns the ordered rsync deploy stages: one stage per content dir
// (ordered so that content that is linked to uploads before the content that links to it,
// avoiding broken links mid-deploy), followed by a catch-all stage for everything else
// that excludes the dirs already handled by the dedicated stages
func buildDeployOptions(source string, destination string, destPathSeparator rune) []deployOptions {
	stageDirNames := []string{
		mediaDirName,
		deployPageDirName,
		deployPostDirName,
		deployPostsDirName,
		deployTagsDirName,
		deployCollectionsDirName,
		deployArchiveDirName,
	}
	deployOpts := make([]deployOptions, 0, len(stageDirNames)+1)
	for _, dirName := range stageDirNames {
		deployOpts = append(deployOpts, deployOptions{
			source:      fmt.Sprintf("%s%c%s%c", source, os.PathSeparator, dirName, os.PathSeparator),
			destination: fmt.Sprintf("%s%c%s", destination, destPathSeparator, dirName),
		})
	}
	return append(deployOpts, deployOptions{
		source:      fmt.Sprintf("%s%c", source, os.PathSeparator),
		destination: destination,
		exclude:     stageDirNames,
	})
}

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

		deployOpts := buildDeployOptions(source, destination, destPathSeparator)

		for i := 0; i < len(deployOpts); i++ {
			dOpts := deployOpts[i]
			if dirExists(dOpts.source) {
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
}
