package storage

import (
	"net/http"

	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

type signResponse struct {
	UploadURL      string `json:"upload_url"`
	APIKey         string `json:"api_key"`
	Timestamp      int64  `json:"timestamp"`
	Signature      string `json:"signature"`
	Folder         string `json:"folder"`
	AllowedFormats string `json:"allowed_formats"`
}

type Handler struct {
	cloudinary *Cloudinary
}

func NewHandler(cloudinary *Cloudinary) *Handler {
	return &Handler{cloudinary: cloudinary}
}

func (h *Handler) Sign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Folder string `json:"folder"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	s, err := h.cloudinary.Sign(req.Folder)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, signResponse{
		UploadURL:      s.UploadURL,
		APIKey:         s.APIKey,
		Timestamp:      s.Timestamp,
		Signature:      s.Signature,
		Folder:         s.Folder,
		AllowedFormats: s.AllowedFormats,
	})
}
