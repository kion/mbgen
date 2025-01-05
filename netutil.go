package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-getter"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed inject-js/admin.js
var adminJS string

//go:embed inject-js/easymde.min.js
var easyMdeJS string

//go:embed inject-js/watch-reload.js
var watchReloadJS string

//go:embed inject-css/easymde.min.css
var easyMdeCSS string

func listenAndServe(addr string, admin bool, watch chan watchReloadData, config appConfig, resLoader resourceLoader) {
	if !dirExists(deployDirName) {
		exitWithError(deployDirName + " directory not found")
	}
	if watch != nil {
		var wsUpgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		watchReloadJS = strings.Replace(watchReloadJS, websocketProtocol, websocketProtocol+addr+websocketPath, 1)
		http.HandleFunc(websocketPath, func(writer http.ResponseWriter, request *http.Request) {
			conn, err := wsUpgrader.Upgrade(writer, request, nil)
			check(err)
			sprintln(" - [reload] websocket connection established")
			pingTicker := time.NewTicker(websocketPingPeriod)
			closed := false
			go func() {
				defer func() {
					pingTicker.Stop()
					err = conn.Close()
					check(err)
					closed = true
				}()
				for {
					_, _, err = conn.ReadMessage()
					if err != nil {
						if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
							sprintln(" - [reload] websocket connection closed by client")
						} else {
							sprintln(" - [reload] error while reading websocket message: ", err)
						}
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
						var msg []byte
						msg, err = json.Marshal(wrd)
						check(err)
						if err = conn.WriteMessage(websocket.TextMessage, msg); err != nil {
							sprintln(" - [reload] error while sending websocket message: ", err)
						} else {
							sprintln(" - [reload] websocket message sent: " + string(msg) + "\n")
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
			html := string(data)
			if strings.HasSuffix(filePath, contentFileExtension) {
				if admin {
					html = strings.Replace(html,
						bodyClosingTag,
						jsOpeningTag+adminJS+jsClosingTag+bodyClosingTag,
						1)
					html = strings.Replace(html,
						bodyClosingTag,
						jsOpeningTag+easyMdeJS+jsClosingTag+bodyClosingTag,
						1)
					html = strings.Replace(html,
						headClosingTag,
						styleOpeningTag+easyMdeCSS+styleClosingTag+bodyClosingTag,
						1)
				}
				if watch != nil {
					html = strings.Replace(html,
						bodyClosingTag,
						jsOpeningTag+watchReloadJS+jsClosingTag+bodyClosingTag,
						1)
				}
			}
			_, err = writer.Write([]byte(html))
			check(err)
		}
	})
	if admin {
		http.HandleFunc("/admin-create", func(writer http.ResponseWriter, request *http.Request) {
			if request.Method == http.MethodPost {
				entryType := request.URL.Query().Get("type")
				entryId := request.URL.Query().Get("id")
				if entryId == "" {
					http.Error(writer, "ID is required", http.StatusBadRequest)
				} else {
					mdEntryPath := fmt.Sprintf("%s%c%s", entryType+"s", os.PathSeparator, entryId+markdownFileExtension)
					if fileExists(mdEntryPath) {
						http.Error(writer, "already exists", http.StatusConflict)
					} else {
						content := "---\n"
						if entryType == "post" {
							content += fmt.Sprintf("date: %s\n", time.Now().Format(time.DateOnly))
							content += fmt.Sprintf("time: %s\n", time.Now().Format(time.TimeOnly))
						}
						content += "title: New " + entryType + "\n"
						content += "\n---\n\n"
						writeDataToFile(mdEntryPath, []byte(content))
						processAndHandleStats(config, resLoader, true)
						contentEntryRedirectURI := fmt.Sprintf("/%s/%s%s", entryType, entryId, contentFileExtension)
						writer.Header().Set("Location", contentEntryRedirectURI)
						writer.WriteHeader(http.StatusCreated)
					}
				}
			}
		})
		http.HandleFunc("/admin-edit", func(writer http.ResponseWriter, request *http.Request) {
			entryType := request.URL.Query().Get("type")
			entryId := request.URL.Query().Get("id")
			mdEntryPath := fmt.Sprintf("%s%c%s", entryType+"s", os.PathSeparator, entryId+markdownFileExtension)
			if request.Method == http.MethodGet {
				mdContent := readDataFromFile(mdEntryPath)
				_, err := writer.Write(mdContent)
				check(err)
			} else if request.Method == http.MethodPost {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					http.Error(writer, "Failed to read request body", http.StatusInternalServerError)
					return
				}
				writeDataToFile(mdEntryPath, body)
				processAndHandleStats(config, resLoader, true)
				contentEntryPath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, entryType, os.PathSeparator, entryId+contentFileExtension)
				content := readDataFromFile(contentEntryPath)
				content = content[strings.Index(string(content), mainOpeningTag)+len(mainOpeningTag):]
				content = content[:strings.Index(string(content), mainClosingTag)]
				_, err = writer.Write(content)
			}
		})
		http.HandleFunc("/admin-delete", func(writer http.ResponseWriter, request *http.Request) {
			entryType := request.URL.Query().Get("type")
			entryId := request.URL.Query().Get("id")
			mdEntryPath := fmt.Sprintf("%s%c%s", entryType+"s", os.PathSeparator, entryId+markdownFileExtension)
			if fileExists(mdEntryPath) {
				// ==================================================
				// delete the markdown file
				// ==================================================
				deleteFile(mdEntryPath)
				processAndHandleStats(config, resLoader, true)
				// ==================================================
				// delete the content file
				// ==================================================
				contentEntryPath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, entryType, os.PathSeparator, entryId+contentFileExtension)
				deleteIfExists(contentEntryPath)
				// ==================================================
				// delete the media directory
				// ==================================================
				mediaDir := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, entryId)
				deleteIfExists(mediaDir)
				// ==================================================
				// delete tag files for the no longer referenced tags
				// ==================================================
				_cleanup(config, commandCleanupTargetTags)
				// ==================================================
				writer.WriteHeader(http.StatusNoContent)
			} else {
				http.Error(writer, "Not found: "+entryType+"/"+entryId, http.StatusNotFound)
			}
		})
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
