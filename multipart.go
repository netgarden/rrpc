package rrpc

import (
	"fmt"
	"net/textproto"
	"strings"
)

// FormDataPartHeader returns the MIME header a multipart bytes field's part
// should carry: the same Content-Disposition mime/multipart.Writer.
// CreateFormFile builds internally, but with contentType instead of its
// hardcoded "application/octet-stream" default - for generated clients that
// need to declare that part's own Content-Type (see @RequestContentType on a
// multipart method). Pass it to (*multipart.Writer).CreatePart in place of
// CreateFormFile.
func FormDataPartHeader(fieldname, filename, contentType string) textproto.MIMEHeader {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(fieldname), escapeQuotes(filename)))
	h.Set("Content-Type", contentType)
	return h
}

// escapeQuotes mirrors mime/multipart's own unexported helper of the same
// name (writer.go), so Content-Disposition values are escaped identically to
// what CreateFormFile would have produced.
func escapeQuotes(s string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
}
