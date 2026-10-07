package ops

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

const maxUploadBytes = 10 << 20 // 10 MiB

// allowedImageExts is the authoritative allow-list. The extension decides the
// stored content type; the client-declared type is only cross-checked. SVG is
// deliberately absent: it can carry script when opened directly, and uploads
// are served from the storefront origin.
var allowedImageExts = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
}

// UploadImage stores an image through the ObjectStorage port. Because storage
// is abstracted, uploads go to S3/OSS in production and to disk in development
// without a handler change.
func (h *Handler) UploadImage(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, fmt.Errorf("%w: file is required", domain.ErrInvalidArgument))
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxUploadBytes {
		response.Fail(c, fmt.Errorf("%w: file must be between 1 byte and 10MiB", domain.ErrInvalidArgument))
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, err)
		return
	}
	defer f.Close()

	// The extension is the allow-list; anything else (including .html and .svg)
	// is rejected so an upload can never be served as an executable document.
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	contentType, ok := allowedImageExts[ext]
	if !ok {
		response.Fail(c, fmt.Errorf("%w: only JPEG, PNG, WebP and GIF images are allowed", domain.ErrInvalidArgument))
		return
	}

	// Sniff the leading bytes and require them to match the claimed type, so a
	// renamed file cannot smuggle a different format past the extension check.
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		response.Fail(c, err)
		return
	}
	if detected := http.DetectContentType(head[:n]); detected != contentType {
		response.Fail(c, fmt.Errorf("%w: file content is %s, not %s", domain.ErrInvalidArgument, detected, contentType))
		return
	}
	body := io.MultiReader(bytes.NewReader(head[:n]), f)

	key := "uploads/" + h.IDs.NewID() + ext
	url, err := h.Storage.Put(c.Request.Context(), key, body, fileHeader.Size, contentType)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "upload.create", "upload", key, nil)
	response.Created(c, gin.H{"key": key, "url": url})
}
