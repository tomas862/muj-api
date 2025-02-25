package utils

import (
	"path/filepath"
	"runtime"
)

func GetCurrentDirectory(depth int) string {
	_, filename, _, _ := runtime.Caller(depth)
	return filepath.Dir(filename)
}

func GetAbsolutePath(relativePath string) string {
	return filepath.Join(GetCurrentDirectory(2), relativePath)
}