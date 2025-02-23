package main

import (
	"bytes"
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
						jsOpeningTag+mdEditorJS+jsClosingTag+bodyClosingTag,
						1)
					html = strings.Replace(html,
						headClosingTag,
						styleOpeningTag+mdEditorCSS+styleClosingTag+bodyClosingTag,
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
					return
				} else {
					mdEntryPath := fmt.Sprintf("%s%c%s", entryType+"s", os.PathSeparator, entryId+markdownFileExtension)
					if fileExists(mdEntryPath) {
						http.Error(writer, "already exists", http.StatusConflict)
						return
					} else {
						content := "---\n"
						if entryType == "post" {
							content += fmt.Sprintf("date: %s\n", time.Now().Format(time.DateOnly))
							content += fmt.Sprintf("time: %s\n", time.Now().Format(time.TimeOnly))
						}
						content += "title: New " + entryType + " title\n"
						content += "\n---\n\n"
						content += "New " + entryType + " content\n\n"
						content += "{media}"
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
				check(err)
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
				return
			}
		})
		http.HandleFunc("/admin-media", func(writer http.ResponseWriter, request *http.Request) {
			entryId := request.URL.Query().Get("id")
			regenerateEntry := false
			if request.Method == http.MethodGet {
				mediaDirPath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, entryId)
				if dirExists(mediaDirPath) {
					mediaFileNames := listAllMedia(entryId, nil)
					listMediaResponse(writer, mediaFileNames, entryId, config, resLoader)
				}
			} else if request.Method == http.MethodPost {
				upMediaFile, upMediaFileHeader, _ := request.FormFile("admin-media-upload-file")
				closeFile(upMediaFile)
				mediaDirPath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, entryId)
				createDirIfNotExists(mediaDirPath)
				mediaFilePath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, entryId, os.PathSeparator, upMediaFileHeader.Filename)
				mediaFile, err := os.Create(mediaFilePath)
				defer closeFile(mediaFile)
				if err == nil {
					_, err = io.Copy(mediaFile, upMediaFile)
				}
				if err != nil {
					printErr(err)
					http.Error(writer, "Failed to upload media file: "+err.Error(), http.StatusInternalServerError)
					return
				} else {
					writer.WriteHeader(http.StatusCreated)
					mediaFileNames := listAllMedia(entryId, nil)
					listMediaResponse(writer, mediaFileNames, entryId, config, resLoader)
					regenerateEntry = true
				}
			} else if request.Method == http.MethodDelete {
				fileName := request.URL.Query().Get("fileName")
				mediaDirPath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, entryId)
				if dirExists(mediaDirPath) {
					mediaFileNames, err := listFilesByExt(mediaDirPath, videoFileExtensions...)
					if err == nil {
						imageFileNames, err := listFilesByExt(mediaDirPath, imageFileExtensions...)
						if err == nil {
							mediaFileNames = append(mediaFileNames, imageFileNames...)
						}
					}
					if err != nil {
						printErr(err)
						http.Error(writer, "Failed to list media files", http.StatusInternalServerError)
						return
					}
					var removeMediaFileNames []string
					for _, mediaFileName := range mediaFileNames {
						// ============================================================
						// `strings.HasPrefix` is being used here to ensure that
						// both the original media file and its thumbnails are deleted
						// ============================================================
						if strings.HasPrefix(mediaFileName, fileName) {
							mediaFilePath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, entryId, os.PathSeparator, mediaFileName)
							err := os.Remove(mediaFilePath)
							if err != nil {
								printErr(err)
								http.Error(writer, "Failed to delete media file: "+mediaFileName, http.StatusInternalServerError)
								return
							}
							removeMediaFileNames = append(removeMediaFileNames, mediaFileName)
						}
					}
					mediaFileNames = removeValuesFromSlice(mediaFileNames, removeMediaFileNames...)
					if len(mediaFileNames) == 0 {
						deleteFile(mediaDirPath)
					}
					writer.WriteHeader(http.StatusResetContent)
					listMediaResponse(writer, mediaFileNames, entryId, config, resLoader)
					regenerateEntry = true
				}
			}
			if regenerateEntry {
				entryType := request.URL.Query().Get("type")
				ceType := contentEntityTypeFromString(entryType)
				removeContentEntityFromCache(ceType, entryId+markdownFileExtension)
				processAndHandleStats(config, resLoader, true)
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

func listMediaResponse(writer http.ResponseWriter, mediaFileNames []string, entryId string, config appConfig, resLoader resourceLoader) {
	allMedia := parseMediaFileNames(mediaFileNames, entryId, config)
	if allMedia != nil {
		inlineMediaTemplate := compileMediaTemplate(resLoader)
		var inlineMediaMarkupBuffer bytes.Buffer
		err := inlineMediaTemplate.Execute(&inlineMediaMarkupBuffer, contentDirectiveData{
			Media: allMedia,
		})
		check(err)
		_, err = writer.Write(inlineMediaMarkupBuffer.Bytes())
		check(err)
	}
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
