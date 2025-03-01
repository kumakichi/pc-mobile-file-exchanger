package handlers

import (
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// UploadHandler handles file upload requests
type UploadHandler struct {
	FS        fs.FS
	BaseURI   string
	UploadDir string
}

// NewUploadHandler creates a new UploadHandler
func NewUploadHandler(fs fs.FS, baseURI, uploadDir string) *UploadHandler {
	return &UploadHandler{
		FS:        fs,
		BaseURI:   baseURI,
		UploadDir: uploadDir,
	}
}

// UploadFormHandler serves the upload form
func (h *UploadHandler) UploadFormHandler(w http.ResponseWriter, _ *http.Request) {
	tmpl, err := template.ParseFS(
		h.FS,
		"templates/base.html",
		"templates/upload.html",
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
	}{
		Title:       "Upload Files",
		GetFiles:    "/file/",
		UploadFiles: "/upload",
		Clipboard:   "/clipboard",
		ToQrcode:    "/qrcode",
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

// UploadHandler processes file uploads
func (h *UploadHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.UploadFormHandler(w, r)
		return
	}

	err := r.ParseMultipartForm(32 << 20) // 32MB max memory
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["uploadFile"]
	if len(files) == 0 {
		http.Error(w, "No files to upload", http.StatusBadRequest)
		return
	}

	okFiles := make([]string, 0, len(files))
	failedFiles := make([]string, 0)

	// Ensure upload directory exists
	if _, err := os.Stat(h.UploadDir); os.IsNotExist(err) {
		err = os.MkdirAll(h.UploadDir, 0755)
		if err != nil {
			http.Error(w, "Failed to create upload directory: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			failedFiles = append(failedFiles, fileHeader.Filename)
			log.Printf("Failed to open uploaded file: %v", err)
			continue
		}
		defer file.Close()

		// Create destination file
		dst, err := os.Create(filepath.Join(h.UploadDir, fileHeader.Filename))
		if err != nil {
			failedFiles = append(failedFiles, fileHeader.Filename)
			log.Printf("Failed to create destination file: %v", err)
			continue
		}
		defer dst.Close()

		// Copy the file content
		_, err = io.Copy(dst, file)
		if err != nil {
			failedFiles = append(failedFiles, fileHeader.Filename)
			log.Printf("Failed to save uploaded file: %v", err)
			continue
		}

		okFiles = append(okFiles, fileHeader.Filename)
	}

	// Show upload result
	tmpl, err := template.ParseFS(
		h.FS,
		"templates/base.html",
		"templates/uploadresult.html",
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
		OkFiles     string
		FailedFiles string
		FilePath    string
	}{
		Title:       "Upload Result",
		GetFiles:    h.BaseURI + "/file/",
		UploadFiles: h.BaseURI + "/upload",
		Clipboard:   h.BaseURI + "/clipboard",
		ToQrcode:    h.BaseURI + "/qrcode",
		OkFiles:     strings.Join(okFiles, ", "),
		FailedFiles: strings.Join(failedFiles, ", "),
		FilePath:    h.UploadDir,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
	}
}
