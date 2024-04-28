package main

import (
	"fmt"
	"path"
	"strings"
	"testing"
)

var subdirs = map[string]string{}

func TestListCommonPrefixes(t *testing.T) {
	var rootPrefix = ""
	if rootPrefix != "" && !strings.HasSuffix(rootPrefix, "/") {
		rootPrefix = rootPrefix + "/"
	}

	subdirs = map[string]string{
		"backups":  path.Join(rootPrefix, "backups") + "/",
		"restores": path.Join(rootPrefix, "restores") + "/",
		"restic":   path.Join(rootPrefix, "restic") + "/",
		"metadata": path.Join(rootPrefix, "metadata") + "/",
		"plugins":  path.Join(rootPrefix, "plugins") + "/",
		"kopia":    path.Join(rootPrefix, "kopia") + "/",
	}

	var dirs []string

	var invalid []string
	for _, dir := range dirs {
		subdir := strings.TrimSuffix(strings.TrimPrefix(dir, rootPrefix), "/")
		if !isValidSubdir(subdir) {
			invalid = append(invalid, subdir)
		}
	}

	if len(invalid) > 0 {
		if len(invalid) > 3 {
			fmt.Printf("Backup store contains %d invalid top-level directories: %v", len(invalid), append(invalid[:3], "..."))
			return
		}
		fmt.Printf("Backup store contains invalid top-level directories: %v", invalid)
	}
}

func isValidSubdir(name string) bool {
	_, ok := subdirs[name]
	return ok
}
