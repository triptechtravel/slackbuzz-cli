package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
)

// CheckUploadFiles fails before anything is posted when a file is missing,
// a directory or empty, so a half-sent message never happens.
func CheckUploadFiles(paths []string) error {
	for _, fp := range paths {
		info, err := os.Stat(fp)
		if err != nil {
			return fmt.Errorf("file not found: %s", fp)
		}
		if info.IsDir() {
			return fmt.Errorf("%s is a directory, not a file", fp)
		}
		if info.Size() == 0 {
			return fmt.Errorf("%s is empty", fp)
		}
	}
	return nil
}

// UploadSpecs turns file paths into upload specs (named by their basename).
func UploadSpecs(paths []string) []slackapi.UploadFileSpec {
	specs := make([]slackapi.UploadFileSpec, len(paths))
	for i, fp := range paths {
		specs[i] = slackapi.UploadFileSpec{FilePath: fp, Filename: filepath.Base(fp)}
	}
	return specs
}
