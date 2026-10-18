package transfer

import (
	"path"
	"strings"
	"unicode"
)

// BundleID is the GADS File Transfer app. The host and the app must agree.
const BundleID = "com.gads.filetransfer"

// StatusName is written inside the app's Documents folder, which is what
// the Files app shows. A leading dot keeps it out of the way.
const StatusName = ".gads-file-transfer.json"

// SafeName keeps the base file name and rejects path tricks.
func SafeName(name string) (string, error) {
	base := path.Base(strings.ReplaceAll(name, "\\", "/"))
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == ".." || base == StatusName {
		return "", errBadName
	}
	if strings.ContainsAny(base, "/\\") || strings.ContainsRune(base, 0) {
		return "", errBadName
	}
	for _, r := range base {
		if unicode.IsControl(r) {
			return "", errBadName
		}
	}
	if len(base) > 200 {
		return "", errBadName
	}
	return base, nil
}

// Kind says where a file should show up on the phone.
type Kind string

const (
	KindMedia Kind = "media"
	KindFile  Kind = "file"
)

// Classify reports media when Photos can store that extension.
func Classify(name string) Kind {
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".heic", ".heif", ".gif", ".tif", ".tiff", ".bmp", ".mov", ".mp4", ".m4v":
		return KindMedia
	default:
		return KindFile
	}
}

var errBadName = badName{}

type badName struct{}

func (badName) Error() string { return "file name is not usable on the phone" }
