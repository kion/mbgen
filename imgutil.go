package main

import (
	"fmt"
	"github.com/disintegration/imaging"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func processImgThumbnails(imgDirPath string, config appConfig) {
	if dirExists(imgDirPath) && config.useThumbs {
		// ==================================================
		// delete any old / no longer needed thumbnails
		// ==================================================
		imgFiles := listFilesByExt(imgDirPath, thumbImageFileExtensions...)
		if len(imgFiles) > 0 {
			for _, imgFile := range imgFiles {
				thm := thumbImgFileNameRegexp.FindStringSubmatch(imgFile)
				if thm != nil {
					thSize, err := strconv.Atoi(thm[1])
					check(err)
					if !slices.Contains(config.thumbSizes, thSize) {
						thumbFilePath := fmt.Sprintf("%s%c%s", imgDirPath, os.PathSeparator, imgFile)
						deleteFile(thumbFilePath)
						sprintln(" - deleted an old / no longer needed thumbnail: " + thumbFilePath)
					}
				}
			}
		}
		// ==================================================
		// generate thumbnails
		// ==================================================
		imgFiles = listFilesByExt(imgDirPath, thumbImageFileExtensions...)
		for _, imgFile := range imgFiles {
			if strings.Contains(imgFile, thumbImgFileSuffix) {
				continue
			}
			imgFilePath := fmt.Sprintf("%s%c%s", imgDirPath, os.PathSeparator, imgFile)
			imgFileExt := filepath.Ext(imgFilePath)
			imgFileSizeInMb := -1.0
			for _, thSize := range config.thumbSizes {
				thumbFilePath := imgFilePath + "_" + strconv.Itoa(thSize) + thumbImgFileSuffix + imgFileExt
				if !fileExists(thumbFilePath) {
					if imgFileSizeInMb < 0 {
						var err error
						imgFileSizeInMb, err = getFileSizeInMb(imgFilePath)
						if err != nil {
							println(" - error reading image file info before thumbnail generation: "+imgFilePath, err)
							continue
						}
					}
					if imgFileSizeInMb >= config.thumbThreshold {
						srcImg, err := imaging.Open(imgFilePath)
						if err != nil {
							println(" - error opening image for thumbnail generation: "+imgFilePath, err)
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
							err = imaging.Save(thImg, thumbFilePath)
							if err != nil {
								println(" - error generating a thumbnail for image: "+imgFilePath, err)
								continue
							}
							println(
								" - generated a thumbnail: "+thumbFilePath,
								" - original image: "+imgFilePath,
								fmt.Sprintf(" - original image dimensions: %dx%d, thumbnail dimensions: %dx%d", iw, ih, tw, th),
							)
							thumbFileSizeInMb, err := getFileSizeInMb(thumbFilePath)
							if err != nil {
								println(" - error reading thumbnail file info: "+thumbFilePath, err)
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
		imgFiles := listFilesByExt(imgDirPath, thumbImageFileExtensions...)
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
