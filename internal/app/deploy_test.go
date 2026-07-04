package app

import (
	"slices"
	"strings"
	"testing"
)

// upload order must ensure no broken links mid-deploy: content that is linked to
// uploads before the content that links to it, with a final catch-all excluding
// every dir already handled by a dedicated stage
func TestBuildDeployOptions(t *testing.T) {
	opts := buildDeployOptions("/src/deploy", "user@host:/var/www", '/')

	expectedStageOrder := []string{
		mediaDirName,
		deployPageDirName,
		deployPostDirName,
		deployPostsDirName,
		deployTagsDirName,
		deployCollectionsDirName,
		deployArchiveDirName,
	}
	if len(opts) != len(expectedStageOrder)+1 {
		t.Fatalf("expected %d deploy stages, got %d", len(expectedStageOrder)+1, len(opts))
	}
	for i, dirName := range expectedStageOrder {
		if !strings.HasSuffix(opts[i].source, "/"+dirName+"/") {
			t.Errorf("stage %d: expected source dir %q, got %q", i, dirName, opts[i].source)
		}
		if !strings.HasSuffix(opts[i].destination, "/"+dirName) {
			t.Errorf("stage %d: expected destination dir %q, got %q", i, dirName, opts[i].destination)
		}
		if len(opts[i].exclude) != 0 {
			t.Errorf("stage %d (%s): dedicated stages must not have excludes, got %v", i, dirName, opts[i].exclude)
		}
	}

	// the catch-all stage must exclude every dir already handled by a dedicated stage
	catchAll := opts[len(opts)-1]
	if catchAll.destination != "user@host:/var/www" {
		t.Errorf("catch-all stage: expected destination %q, got %q", "user@host:/var/www", catchAll.destination)
	}
	for _, dirName := range expectedStageOrder {
		if !slices.Contains(catchAll.exclude, dirName) {
			t.Errorf("catch-all stage: missing exclude for %q (excludes: %v)", dirName, catchAll.exclude)
		}
	}
}
