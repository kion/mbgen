package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-getter"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed ws-watch-reload.html
var wsHtml string

const (
	pingPeriod = 60 * time.Second
)

func listenAndServe(addr string, watch chan watchReloadData) {
	if watch != nil {
		var wsUpgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		wsHtml = strings.Replace(wsHtml, websocketProtocol, websocketProtocol+addr+websocketPath, 1) + contentClosingTag
		http.HandleFunc(websocketPath, func(writer http.ResponseWriter, request *http.Request) {
			conn, err := wsUpgrader.Upgrade(writer, request, nil)
			check(err)
			sprintln(" - [reload] websocket connection established")
			pingTicker := time.NewTicker(pingPeriod)
			closed := false
			go func() {
				for {
					_, _, err = conn.ReadMessage()
					if err != nil {
						if websocket.IsCloseError(err, websocket.CloseGoingAway) {
							sprintln(" - [reload] websocket connection closed by client")
						} else {
							sprintln(" - [reload] error while reading websocket message: ", err)
						}
						closed = true
						pingTicker.Stop()
						break
					}
				}
			}()
			for {
				if closed {
					break
				}
				select {
				case wrd, ok := <-watch:
					if ok && !closed {
						var msg string
						if wrd.Deleted {
							msg = `{ "type": "{{type}}", "id": "{{id}}", "deleted": true }`
						} else {
							msg = `{ "type": "{{type}}", "id": "{{id}}" }`
						}
						msg = strings.ReplaceAll(msg, "{{type}}", strings.ToLower(wrd.Type.String()))
						msg = strings.ReplaceAll(msg, "{{id}}", wrd.Id)
						if err = conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
							sprintln(" - [reload] error while sending websocket message: ", err)
						} else {
							sprintln(" - [reload] websocket message sent: " + msg + "\n")
						}
					} else {
						break
					}
				case <-pingTicker.C:
					if !closed {
						if err = conn.WriteMessage(websocket.TextMessage, []byte{}); err != nil {
							sprintln(" - [ping] error while sending websocket message: ", err)
						} else {
							sprintln(" - [ping] websocket message sent")
						}
					} else {
						break
					}
				}
			}
		})
	}
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		path := request.URL.Path
		println(" - request received: " + path)
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
			if watch != nil && strings.HasSuffix(filePath, contentFileExtension) {
				html := strings.Replace(string(data),
					contentClosingTag,
					wsHtml,
					1)
				data = []byte(html)
			}
			_, err = writer.Write(data)
			check(err)
		}
	})
	if !dirExists(deployDirName) {
		exitWithError(deployDirName + " directory not found")
	}
	url := addr
	if strings.Contains(url, "localhost") {
		url = httpProtocol + url
	} else {
		url = httpsProtocol + url
	}
	sprintln(
		"[ ----- serving ------ ]\n",
		" - "+url+"\n",
	)
	err := http.ListenAndServe(addr, nil)
	exitWithError(err.Error())
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
