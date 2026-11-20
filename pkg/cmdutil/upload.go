package cmdutil

import (
	"fmt"
	"os"
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
