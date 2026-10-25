package controllers

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"transcendance/models"
	"transcendance/views"

	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (pc *ProfileController) AvatarGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.Status(http.StatusNotFound)
		return
	}

	var user models.User
	err = pc.DB.Select("avatar").First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusNotFound)
		} else {
			c.Status(http.StatusInternalServerError)
		}
		return
	}
	data, contentType := user.Avatar, "image/webp"
	if len(data) < 12 || !bytes.Equal(data[:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WEBP")) {
		data, err = os.ReadFile("static/profile-picture.svg")
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		contentType = "image/svg+xml"
	}
	serveAvatar(c.Writer, c.Request, data, contentType)
}

func (pc *ProfileController) avatarError(c *gin.Context, status int, message string) {
	c.Header("Cache-Control", "no-store")
	pc.Renderer.RenderError(c, views.ErrorPage{
		User:       GetAuthenticatedUser(c),
		StatusCode: status,
		Message:    pc.Renderer.T(c, message),
	})
}

func (pc *ProfileController) avatarOwner(c *gin.Context) *AuthenticatedUser {
	user := GetAuthenticatedUser(c)
	if user == nil {
		pc.avatarError(c, http.StatusUnauthorized, "error-unauthorized")
		return nil
	}
	origin := c.GetHeader("Origin")
	if origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != c.Request.Host || (u.Scheme != "http" && u.Scheme != "https") {
			pc.avatarError(c, http.StatusForbidden, "error-unauthorized")
			return nil
		}
	}
	if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
		pc.avatarError(c, http.StatusForbidden, "error-unauthorized")
		return nil
	}
	return user
}

func (pc *ProfileController) AvatarPost(c *gin.Context) {
	user := pc.avatarOwner(c)
	if user == nil {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarBytes+64*1024)
	err := c.Request.ParseMultipartForm(maxAvatarBytes + 64*1024)
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		status := http.StatusBadRequest
		if errors.As(err, &tooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		pc.avatarError(c, status, "profile-avatar-invalid")
		return
	}

	files := c.Request.MultipartForm.File["avatar"]
	if len(files) != 1 || files[0].Size > maxAvatarBytes {
		pc.avatarError(c, http.StatusBadRequest, "profile-avatar-invalid")
		return
	}

	f, err := files[0].Open()
	if err != nil {
		pc.avatarError(c, http.StatusBadRequest, "profile-avatar-invalid")
		return
	}
	defer f.Close()

	input, err := io.ReadAll(io.LimitReader(f, maxAvatarBytes+1))
	if err != nil {
		pc.avatarError(c, http.StatusBadRequest, "profile-avatar-invalid")
		return
	}

	data, err := normalizeAvatar(input)
	if err != nil {
		pc.avatarError(c, http.StatusBadRequest, "profile-avatar-invalid")
		return
	}
	pc.saveAvatar(c, user.ID, data)
}

func (pc *ProfileController) AvatarDelete(c *gin.Context) {
	user := pc.avatarOwner(c)
	if user == nil {
		return
	}
	pc.saveAvatar(c, user.ID, nil)
}

func (pc *ProfileController) saveAvatar(c *gin.Context, userID uint, data []byte) {
	if err := pc.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar", data).Error; err != nil {
		pc.avatarError(c, http.StatusInternalServerError, "error-unexpected")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusSeeOther, "/profile")
}

const maxAvatarBytes = 5 * 1024 * 1024

func normalizeAvatar(input []byte) ([]byte, error) {
	if len(input) == 0 || len(input) > maxAvatarBytes {
		return nil, fmt.Errorf("image must be at most 5 MiB")
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(input))
	if err != nil || (format != "jpeg" && format != "png" && format != "webp") {
		return nil, fmt.Errorf("expected a JPEG, PNG or WebP image")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 1024 || cfg.Height > 1024 {
		return nil, fmt.Errorf("image dimensions exceed the limit")
	}

	m, _, err := image.Decode(bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}

	var out bytes.Buffer
	if err := webp.Encode(&out, m, &webp.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode WebP: %w", err)
	}
	if out.Len() > maxAvatarBytes {
		return nil, fmt.Errorf("converted image exceeds 5 MiB")
	}
	return out.Bytes(), nil
}

func serveAvatar(w http.ResponseWriter, r *http.Request, data []byte, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, no-cache")
	w.Header().Set("ETag", fmt.Sprintf(`"%x"`, sha256.Sum256(data)))
	http.ServeContent(w, r, "avatar", time.Time{}, bytes.NewReader(data))
}
