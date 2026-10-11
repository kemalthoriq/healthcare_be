package storage

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

const (
	rootFolder        = "lumiere"
	allowedFormats    = "jpg,png,webp"
	maxImageURLLength = 500
)

var folders = map[string]bool{"treatments": true, "therapists": true, "promos": true}

var (
	ErrNotConfigured = apperr.Unavailable("UPLOAD_NOT_CONFIGURED", "Upload gambar belum diaktifkan di server.")
	ErrInvalidFolder = apperr.Validation(map[string]string{"folder": "Folder harus treatments, therapists, atau promos."})
)

const (
	msgNotHTTPS      = "URL gambar harus diawali https:// dan maksimal 500 karakter."
	msgNotCloudinary = "Gambar harus diunggah lewat fitur upload klinik."
)

type Signature struct {
	UploadURL      string
	APIKey         string
	Timestamp      int64
	Signature      string
	Folder         string
	AllowedFormats string
}

type Cloudinary struct {
	cloudName string
	apiKey    string
	apiSecret string
	now       func() time.Time
}

func NewCloudinary(cloudName, apiKey, apiSecret string) *Cloudinary {
	return &Cloudinary{cloudName: cloudName, apiKey: apiKey, apiSecret: apiSecret, now: time.Now}
}

func (c *Cloudinary) Configured() bool {
	return c.cloudName != "" && c.apiKey != "" && c.apiSecret != ""
}

func (c *Cloudinary) Sign(folder string) (Signature, error) {
	if !c.Configured() {
		return Signature{}, ErrNotConfigured
	}
	if !folders[folder] {
		return Signature{}, ErrInvalidFolder
	}

	timestamp := c.now().Unix()
	params := map[string]string{
		"allowed_formats": allowedFormats,
		"folder":          rootFolder + "/" + folder,
		"timestamp":       strconv.FormatInt(timestamp, 10),
	}
	return Signature{
		UploadURL:      fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", c.cloudName),
		APIKey:         c.apiKey,
		Timestamp:      timestamp,
		Signature:      sign(params, c.apiSecret),
		Folder:         params["folder"],
		AllowedFormats: allowedFormats,
	}, nil
}

func (c *Cloudinary) CheckImageURL(raw string) string {
	if raw == "" {
		return ""
	}
	if len(raw) > maxImageURLLength {
		return msgNotHTTPS
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return msgNotHTTPS
	}
	if !c.Configured() {
		return ""
	}

	prefix := "/" + c.cloudName + "/image/upload/"
	if u.Host != "res.cloudinary.com" || !strings.HasPrefix(u.Path, prefix) || !strings.Contains(u.Path, "/"+rootFolder+"/") ||
		u.RawQuery != "" || u.Fragment != "" {
		return msgNotCloudinary
	}
	return ""
}

func sign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+params[k])
	}
	sum := sha1.Sum([]byte(strings.Join(pairs, "&") + secret))
	return hex.EncodeToString(sum[:])
}
