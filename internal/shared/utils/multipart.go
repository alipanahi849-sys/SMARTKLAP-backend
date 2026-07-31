package utils

import (
	"mime/multipart"

	"github.com/gin-gonic/gin"
)

// FirstFormFile returns the first multipart file matching any of the given
// field names, or the first uploaded file in the form as a fallback.
func FirstFormFile(c *gin.Context, names ...string) *multipart.FileHeader {
	for _, name := range names {
		if file, err := c.FormFile(name); err == nil && file != nil {
			return file
		}
	}

	form, err := c.MultipartForm()
	if err != nil || form == nil {
		return nil
	}
	for _, files := range form.File {
		if len(files) > 0 && files[0] != nil {
			return files[0]
		}
	}
	return nil
}
