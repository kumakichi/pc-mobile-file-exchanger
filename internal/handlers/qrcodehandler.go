package handlers

import (
	"encoding/base64"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"

	"github.com/skip2/go-qrcode"
)

// QRCodeHandler handles QR code generation requests
type QRCodeHandler struct {
	FS      fs.FS
	BaseURI string
	pattern string
}

// NewQRCodeHandler creates a new QRCodeHandler
func NewQRCodeHandler(fs fs.FS, baseURI, pattern string) *QRCodeHandler {
	return &QRCodeHandler{
		FS:      fs,
		BaseURI: baseURI,
		pattern: pattern,
	}
}

// QRCodeHandler generates and serves a QR code
func (h *QRCodeHandler) QRCodeHandler(w http.ResponseWriter, _ *http.Request) {
	u, err := url.JoinPath(h.BaseURI, h.pattern)
	if err != nil {
		http.Error(w, "Failed to join URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	qr, err := qrcode.New(u, qrcode.Medium)
	if err != nil {
		http.Error(w, "Failed to generate QR code: "+err.Error(), http.StatusInternalServerError)
		return
	}

	png, err := qr.PNG(256)
	if err != nil {
		http.Error(w, "Failed to encode QR code as PNG: "+err.Error(), http.StatusInternalServerError)
		return
	}

	base64Str := base64.StdEncoding.EncodeToString(png)

	tmpl, err := template.ParseFS(
		h.FS,
		"templates/base.html",
		"templates/qrcode.html",
	)
	if err != nil {
		http.Error(w, "Failed to parse template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title       string
		GetFiles    string
		UploadFiles string
		Clipboard   string
		ToQrcode    string
		QrBase      string
	}{
		Title:       "QR Code",
		GetFiles:    "/file/",
		UploadFiles: "/upload",
		Clipboard:   "/clipboard",
		ToQrcode:    "/qrcode",
		QrBase:      base64Str,
	}

	if h.BaseURI != "" {
		data.GetFiles = h.BaseURI + "/file/"
		data.UploadFiles = h.BaseURI + "/upload"
		data.Clipboard = h.BaseURI + "/clipboard"
		data.ToQrcode = h.BaseURI + "/qrcode"
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
	}
}
