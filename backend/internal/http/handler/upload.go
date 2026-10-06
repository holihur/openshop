package handler

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

const maxUploadBytes = 10 << 20 // 10 MiB

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
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
	if fileHeader.Size > maxUploadBytes {
		response.Fail(c, fmt.Errorf("%w: file exceeds 10MiB", domain.ErrInvalidArgument))
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, err)
		return
	}
	defer f.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	ext, ok := allowedImageTypes[contentType]
	if !ok {
		ext = strings.ToLower(filepath.Ext(fileHeader.Filename))
		if ext == "" {
			response.Fail(c, fmt.Errorf("%w: unsupported image type", domain.ErrInvalidArgument))
			return
		}
	}

	key := "uploads/" + h.IDs.NewID() + ext
	url, err := h.Storage.Put(c.Request.Context(), key, f, fileHeader.Size, contentType)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, gin.H{"key": key, "url": url})
}
