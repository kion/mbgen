package app

import (
	"fmt"
	"github.com/disintegration/imaging"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func processImgThumbnails(mediaDirPath string, config appConfig) {
	if dirExists(mediaDirPath) && config.useThumbs {
		// ==================================================
		// delete any old / no longer needed thumbnails
		// ==================================================
		imgFiles, err := listFilesByExt(mediaDirPath, thumbImageFileExtensions...)
		check(err)
		if len(imgFiles) > 0 {
			for _, imgFile := range imgFiles {
				thm := thumbImgFileNameRegexp.FindStringSubmatch(imgFile)
				if thm != nil {
					thSize, err := strconv.Atoi(thm[1])
					check(err)
					if !slices.Contains(config.thumbSizes, thSize) {
						thumbFilePath := fmt.Sprintf("%s%c%s", mediaDirPath, os.PathSeparator, imgFile)
						deleteFile(thumbFilePath)
						sprintln(" - deleted an old / no longer needed thumbnail: " + thumbFilePath)
					}
				}
			}
		}
		// ==================================================
		// generate thumbnails
		// ==================================================
		imgFiles, err = listFilesByExt(mediaDirPath, thumbImageFileExtensions...)
		check(err)
		for _, imgFile := range imgFiles {
			if strings.Contains(imgFile, thumbImgFileSuffix) {
				continue
			}
			imgFilePath := fmt.Sprintf("%s%c%s", mediaDirPath, os.PathSeparator, imgFile)
			imgFileExt := filepath.Ext(imgFilePath)
			imgFileSizeInMb := -1.0
			for _, thSize := range config.thumbSizes {
				thumbFilePath := imgFilePath + "_" + strconv.Itoa(thSize) + thumbImgFileSuffix + imgFileExt
				if !fileExists(thumbFilePath) {
					if imgFileSizeInMb < 0 {
						var err error
						imgFileSizeInMb, err = getFileSizeInMb(imgFilePath)
						if err != nil {
							sprintln(" - error reading image file info before thumbnail generation: "+imgFilePath, err)
							continue
						}
					}
					if imgFileSizeInMb >= config.thumbThreshold {
						srcImg, err := imaging.Open(imgFilePath)
						if err != nil {
							sprintln(" - error opening image for thumbnail generation: "+imgFilePath, err)
							continue
						}
						iw := srcImg.Bounds().Dx()
						ih := srcImg.Bounds().Dy()
						if iw > thSize || ih > thSize {
							var tw, th int
							if iw == ih {
								tw = thSize
								th = thSize
							} else if iw > ih {
								tw = thSize
								th = thSize * ih / iw
							} else {
								th = thSize
								tw = thSize * iw / ih
							}
							thImg := imaging.Resize(srcImg, tw, th, imaging.Lanczos)
							if imgFileExt == ".jpg" || imgFileExt == ".jpeg" {
								err = imaging.Save(thImg, thumbFilePath, imaging.JPEGQuality(config.jpegQuality))
							} else if imgFileExt == ".png" {
								err = imaging.Save(thImg, thumbFilePath, imaging.PNGCompressionLevel(config.pngCompressionLevel.Value()))
							}
							if err != nil {
								sprintln(" - error generating a thumbnail for image: "+imgFilePath, err)
								continue
							}
							sprintln(
								" - generated a thumbnail: "+thumbFilePath,
								" - original image: "+imgFilePath,
								fmt.Sprintf(" - original image dimensions: %dx%d, thumbnail dimensions: %dx%d", iw, ih, tw, th),
							)
							thumbFileSizeInMb, err := getFileSizeInMb(thumbFilePath)
							if err != nil {
								sprintln(" - error reading thumbnail file info: "+thumbFilePath, err)
								continue
							}
							println(fmt.Sprintf(" - original image file size: %.2f MB, thumbnail file size: %.2f MB\n", imgFileSizeInMb, thumbFileSizeInMb))
						}
					}
				}
			}
		}
	}
}

func deleteImgThumbnails(imgDirPath string, config appConfig) {
	if dirExists(imgDirPath) && !config.useThumbs {
		imgFiles, err := listFilesByExt(imgDirPath, thumbImageFileExtensions...)
		check(err)
		if len(imgFiles) > 0 {
			for _, imgFile := range imgFiles {
				thm := thumbImgFileNameRegexp.FindStringSubmatch(imgFile)
				if thm != nil {
					thumbFilePath := fmt.Sprintf("%s%c%s", imgDirPath, os.PathSeparator, imgFile)
					deleteFile(thumbFilePath)
					sprintln(" - deleted thumbnail: " + thumbFilePath)
				}
			}
		}
	}
}

func processOriginalMediaFiles(config appConfig, dryRun bool) bool {
	resizeCnt := 0
	if config.resizeOrigImages && config.maxImgSize > 0 {
		deployMediaDir := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, mediaDirName)
		if dirExists(deployMediaDir) {
			ceTypeMediaDirs := []string{
				fmt.Sprintf("%s%c%s", deployMediaDir, os.PathSeparator, deployPageDirName),
				fmt.Sprintf("%s%c%s", deployMediaDir, os.PathSeparator, deployPostDirName),
			}
			if len(ceTypeMediaDirs) > 0 {
				sprintln(" - inspecting original media files ...")
				for _, ceTypeMediaDir := range ceTypeMediaDirs {
					if dirExists(ceTypeMediaDir) {
						mediaDirEntries, err := os.ReadDir(ceTypeMediaDir)
						check(err)
						if len(mediaDirEntries) > 0 {
							for _, mediaDirEntry := range mediaDirEntries {
								mediaDirEntryInfo, err := mediaDirEntry.Info()
								check(err)
								if mediaDirEntryInfo.IsDir() {
									mediaDirEntryName := mediaDirEntryInfo.Name()
									mediaDirPath := fmt.Sprintf("%s%c%s", ceTypeMediaDir, os.PathSeparator, mediaDirEntryName)
									imgFiles, err := listFilesByExt(mediaDirPath, thumbImageFileExtensions...)
									check(err)
									if len(imgFiles) > 0 {
										for _, imgFile := range imgFiles {
											if !strings.Contains(imgFile, thumbImgFileSuffix) { // skip thumbnail files
												imgFilePath := fmt.Sprintf("%s%c%s", mediaDirPath, os.PathSeparator, imgFile)
												if processOriginalMediaFile(imgFilePath, config, dryRun) {
													resizeCnt++
												}
											}
										}
									}
								}
							}
							if resizeCnt > 0 {
								if dryRun {
									sprintln(" - " + strconv.Itoa(resizeCnt) + " original image(s) exceed the max size")
								} else {
									sprintln(" - resized " + strconv.Itoa(resizeCnt) + " original image(s)")
								}
							}
						}
					}
				}
			}
		}
	}
	return resizeCnt > 0
}

func processOriginalMediaFile(mediaFilePath string, config appConfig, dryRun bool) bool {
	if config.resizeOrigImages {
		fileExt := strings.ToLower(filepath.Ext(mediaFilePath))
		if slices.Contains(thumbImageFileExtensions, fileExt) {
			maxImgSize := config.maxImgSize
			if maxImgSize > 0 {
				origImg, err := imaging.Open(mediaFilePath)
				if err != nil {
					sprintln(" - error opening image file for resizing: "+mediaFilePath, err)
				} else {
					ow := origImg.Bounds().Dx()
					oh := origImg.Bounds().Dy()

					if ow > maxImgSize || oh > maxImgSize {
						// ==================================================
						// calculate the new image dimensions
						// ==================================================
						var tw, th int
						if ow == oh {
							tw = maxImgSize
							th = maxImgSize
						} else if ow > oh {
							tw = maxImgSize
							th = maxImgSize * oh / ow
						} else {
							th = maxImgSize
							tw = maxImgSize * ow / oh
						}

						if dryRun {
							// ==================================================
							// report the image file that exceeds the max size
							// ==================================================
							sprintln(
								" - original image exceeds the max size: "+mediaFilePath,
								fmt.Sprintf(" - original image dimensions: %dx%d, expected dimensions: %dx%d", ow, oh, tw, th),
							)
							return true
						} else {
							// ==================================================
							// resize and save the image to the original file
							// ==================================================
							newImg := imaging.Resize(origImg, tw, th, imaging.Lanczos)
							if fileExt == ".jpg" || fileExt == ".jpeg" {
								err = imaging.Save(newImg, mediaFilePath, imaging.JPEGQuality(config.jpegQuality))
							} else if fileExt == ".png" {
								err = imaging.Save(newImg, mediaFilePath, imaging.PNGCompressionLevel(config.pngCompressionLevel.Value()))
							}
							// ==================================================

							if err != nil {
								sprintln(" - error saving resized image: "+mediaFilePath, err)
							} else {
								sprintln(
									" - resized the original image: "+mediaFilePath,
									fmt.Sprintf(" - original image dimensions: %dx%d, resized dimensions: %dx%d", ow, oh, tw, th),
								)
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}
