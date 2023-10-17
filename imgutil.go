package main

import (
	"fmt"
	"github.com/disintegration/imaging"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func processImgThumbnails(imgDirPath string, config appConfig) {
	// ==================================================
	// delete any old / no longer needed thumbnails
	// ==================================================
	imgFiles := listFilesByExtRegexp(imgDirPath, thumbImageFileExtensions)
	if len(imgFiles) > 0 {
		for _, imgFile := range imgFiles {
			thm := thumbImgFileNameRegexp.FindStringSubmatch(imgFile)
			if thm != nil {
				thSize, err := strconv.Atoi(thm[1])
				check(err)
				if !slices.Contains(config.thumbSizes, thSize) {
					imgFilePath := fmt.Sprintf("%s%c%s", imgDirPath, os.PathSeparator, imgFile)
					deleteFile(imgFilePath)
					log.Println(" - deleted an old / no longer needed thumbnail: " + imgFilePath)
				}
			}
		}
	}
	// ==================================================
	// generate thumbnails
	// ==================================================
	imgFiles = listFilesByExtRegexp(imgDirPath, thumbImageFileExtensions)
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
						log.Println(" - error reading image file info before thumbnail generation: " + imgFilePath)
						log.Println(err)
						continue
					}
				}
				if imgFileSizeInMb >= config.thumbThreshold {
					srcImg, err := imaging.Open(imgFilePath)
					if err != nil {
						log.Println(" - error opening image for thumbnail generation: " + imgFilePath)
						log.Println(err)
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
							log.Println(" - error generating a thumbnail for image: " + imgFilePath)
							log.Println(err)
							continue
						}
						log.Println(" - generated a thumbnail: " + thumbFilePath)
						log.Println(" - original image: " + imgFilePath)
						log.Println(fmt.Sprintf(" - original image dimensions: %dx%d, thumbnail dimensions: %dx%d", iw, ih, tw, th))
						thumbFileSizeInMb, err := getFileSizeInMb(thumbFilePath)
						if err != nil {
							log.Println(" - error reading thumbnail file info: " + thumbFilePath)
							log.Println(err)
							continue
						}
						log.Println(fmt.Sprintf(" - original image file size: %.2f MB, thumbnail file size: %.2f MB\n", imgFileSizeInMb, thumbFileSizeInMb))
					}
				}
			}
		}
	}
}
