package embedded

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"resin/pkg/logging"
)

//go:embed assets/*
var AssetFiles embed.FS

//go:embed login/*
var LoginFiles embed.FS

func ReadAssets[T any](a *T) {
	val := reflect.ValueOf(a)
	elem := val.Elem()
	for i := 0; i < elem.NumField(); i++ {
		file, ok := elem.Type().Field(i).Tag.Lookup("asset")
		if !ok {
			continue // no tag
		}
		bytes, err := AssetFiles.ReadFile(fmt.Sprintf("assets/%s", file))
		// Panic on failure to read any asset
		if err != nil {
			logging.Panic("Failed to read assets:\n%v", err)
			os.Exit(1)
			return
		}

		elem.Field(i).SetBytes(bytes)
	}
	return
}

func ExtractEmbeddedFiles() {
	extractDir(LoginFiles, "login", ".")
}

func extractDir(fs embed.FS, srcDir string, destDir string) {
	read, err := fs.ReadDir(srcDir)
	if err != nil {
		logging.Fail("Failed to read asset dir \"%s\":\n%v", srcDir, err)
		return
	}

	targetDir := filepath.Join(destDir, srcDir)
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		logging.Fail("Failed to create dir \"%s\":\n%v", targetDir, err)
		return
	}

	for _, e := range read {
		srcPath := srcDir + "/" + e.Name()
		destPath := filepath.Join(destDir, srcPath)

		if e.IsDir() {
			extractDir(fs, srcPath, destDir)
			continue
		}

		// Force overwrite to ensure we always have the latest version from inside the binary
		file, err := fs.ReadFile(srcPath)
		if err != nil {
			logging.Fail("failed to read file \"%s\":\n%v", srcPath, err)
			continue
		}

		newFile, err := os.Create(destPath)
		if err != nil {
			logging.Fail("failed to create file \"%s\":\n%v", destPath, err)
			continue
		}

		n, err := newFile.Write(file)
		newFile.Close()
		if err != nil {
			logging.Fail("failed to write file \"%s\":\n%v", destPath, err)
			continue
		}
		logging.Info("%s: wrote %d bytes", destPath, n)
	}
}
