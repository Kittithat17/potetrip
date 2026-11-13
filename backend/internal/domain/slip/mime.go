package slip

import (
	"bytes"
	"net/http"
)

// Allowed upload types (spec 6.3).
var allowedTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true, "image/heic": true}

// DetectImageType sniffs the bytes and returns an allowed MIME type, or ""
// if the content is not one of them. The client's declared type is not trusted.
func DetectImageType(b []byte) string {
	if isHEIC(b) {
		return "image/heic"
	}
	t := http.DetectContentType(b)
	if allowedTypes[t] {
		return t
	}
	return ""
}

// isHEIC checks the ISO-BMFF "ftyp" box for HEIF brands.
func isHEIC(b []byte) bool {
	if len(b) < 12 || !bytes.Equal(b[4:8], []byte("ftyp")) {
		return false
	}
	switch string(b[8:12]) {
	case "heic", "heix", "hevc", "hevx", "heim", "heis", "mif1", "msf1":
		return true
	}
	return false
}
