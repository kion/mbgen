package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func listFilesByExt(dir string, extensions ...string) []string {
	var files []string
	extSet := make(map[string]bool)
	for _, v := range extensions {
		extSet[v] = true
	}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() && extSet[filepath.Ext(path)] {
			files = append(files, f.Name())
		}
		return nil
	})
	check(err)
	return files
}

func copyDir(scrDir, dst string) {
	entries, err := os.ReadDir(scrDir)
	check(err)
	for _, entry := range entries {
		sourcePath := filepath.Join(scrDir, entry.Name())
		destPath := filepath.Join(dst, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		check(err)

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			createDirIfNotExists(destPath)
			copyDir(sourcePath, destPath)
		case os.ModeSymlink:
			copySymLink(sourcePath, destPath)
		default:
			copyFile(sourcePath, destPath)
		}

		fInfo, err := entry.Info()
		check(err)

		isSymlink := fInfo.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			err := os.Chmod(destPath, fInfo.Mode())
			check(err)
		}
	}
}

func copyFile(srcFile, dstFile string) {
	out, err := os.Create(dstFile)
	check(err)
	defer closeFile(out)

	in, err := os.Open(srcFile)
	check(err)
	defer closeFile(in)

	_, err = io.Copy(out, in)
	check(err)
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func fileExists(path string) bool {
	if file, err := os.Stat(path); os.IsNotExist(err) || file.IsDir() {
		return false
	}
	return true
}

func dirExists(path string) bool {
	if dir, err := os.Stat(path); os.IsNotExist(err) || !dir.IsDir() {
		return false
	}
	return true
}

func createDirIfNotExists(dir string) {
	if !fileExists(dir) {
		createDir(dir)
	}
}

func createDir(dir string) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		panic(fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error()))
	}
}

func recreateDir(dir string) {
	deleteIfExists(dir)
	createDir(dir)
}

func deleteFile(filePath string) {
	err := os.Remove(filePath)
	check(err)
}

func deleteIfExists(path string) bool {
	if pathExists(path) {
		err := os.RemoveAll(path)
		check(err)
		return true
	}
	return false
}

func copySymLink(source, dest string) {
	link, err := os.Readlink(source)
	check(err)
	err = os.Symlink(link, dest)
	check(err)
}

func closeFile(file *os.File) {
	err := file.Close()
	check(err)
}

func writeDataToFile(outputFilePath string, data []byte) {
	outputFile, err := os.Create(outputFilePath)
	check(err)
	_, err = outputFile.Write(data)
	check(err)
	err = outputFile.Close()
	check(err)
}

func writeDataToFileIfChanged(outputFilePath string, data []byte) bool {
	changed := true
	if fileExists(outputFilePath) {
		existingData, err := os.ReadFile(outputFilePath)
		check(err)
		if bytes.Equal(existingData, data) {
			changed = false
		}
	}
	if changed {
		writeDataToFile(outputFilePath, data)
	}
	return changed
}

func getFileSizeInMb(filePath string) (float64, error) {
	imgFileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	fileSizeInMb := float64(imgFileInfo.Size()) / 1000 / 1000
	return fileSizeInMb, nil
}
