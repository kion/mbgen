package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupTagsPreservesReferencedMultiWordTag(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mbgen-cleanup-tags-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.MkdirAll(markdownPostsDirName, 0o755); err != nil {
		t.Fatal(err)
	}
	postContent := "---\n" +
		"date: 2026-04-18\n" +
		"tags:\n" +
		"  - Multi Word Tag\n" +
		"---\n\n" +
		"Body.\n"
	postPath := filepath.Join(markdownPostsDirName, "sample-post"+markdownFileExtension)
	if err := os.WriteFile(postPath, []byte(postContent), 0o644); err != nil {
		t.Fatal(err)
	}

	referencedTagDir := filepath.Join(deployDirName, deployTagsDirName, "multi-word-tag")
	if err := os.MkdirAll(referencedTagDir, 0o755); err != nil {
		t.Fatal(err)
	}
	unusedTagDir := filepath.Join(deployDirName, deployTagsDirName, "stale-tag")
	if err := os.MkdirAll(unusedTagDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_cleanup(defaultConfig(), commandCleanupTargetTags)

	if !dirExists(referencedTagDir) {
		t.Errorf("referenced multi-word tag dir was incorrectly deleted: %s", referencedTagDir)
	}
	if dirExists(unusedTagDir) {
		t.Errorf("unused tag dir should have been deleted but still exists: %s", unusedTagDir)
	}
}
