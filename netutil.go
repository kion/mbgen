package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/go-getter"
	"log"
	"net/http"
	"os"
	"strings"
)

func listenAndServe(addr string) {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		path := request.URL.Path
		log.Println(" - request received: " + path)
		specificResourceRequested := strings.Contains(path, ".")
		if !specificResourceRequested && path[len(path)-1] != '/' {
			path += "/"
			http.Redirect(writer, request, path, http.StatusFound)
		} else {
			filePath := fmt.Sprintf("%s%s", deployDirName, path)
			if !specificResourceRequested {
				filePath += indexPageFileName
			}
			data, err := os.ReadFile(filePath)
			check(err)
			_, err = writer.Write(data)
			check(err)
		}
	})
	if !dirExists(deployDirName) {
		exitWithError(deployDirName + " directory not found")
	}
	fmt.Println("")
	fmt.Println("[ ----- serving ------ ]")
	fmt.Println("")
	url := addr
	if strings.Contains(url, "localhost") {
		url = "http://" + url
	} else {
		url = "https://" + url
	}
	fmt.Println(" - " + url)
	fmt.Println("")
	err := http.ListenAndServe(addr, nil)
	log.Fatal(err)
}

func download(sourceUrl string, destinationDir string) error {
	client := &getter.Client{
		Ctx:  context.Background(),
		Dst:  destinationDir,
		Dir:  true,
		Src:  sourceUrl,
		Mode: getter.ClientModeDir,
		Detectors: []getter.Detector{
			&getter.GitHubDetector{},
		},
		Getters: map[string]getter.Getter{
			"git": &getter.GitGetter{},
		},
	}
	if err := client.Get(); err != nil {
		return errors.New(fmt.Sprintf("error downloading %s: %v", client.Src, err))
	}
	return nil
}
