package handlers

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
)

// FileHandler handles file-related requests
type FileHandler struct {
	FS            fs.FS
	BaseURI       string
	Directory     string
	FilterSuffix  string
	PatchHTMLFile bool
}

// NewFileHandler creates a new FileHandler
func NewFileHandler(fs fs.FS, baseURI, directory, filterSuffix string, patchHTMLFile bool) *FileHandler {
	return &FileHandler{
		FS:            fs,
		BaseURI:       baseURI,
		Directory:     directory,
		FilterSuffix:  filterSuffix,
		PatchHTMLFile: patchHTMLFile,
	}
}

// WrapFSHandler wraps a filesystem handler with custom behavior
func (h *FileHandler) WrapFSHandler(fileHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		log.Printf("WrapFSHandler called with URL path: %s", urlPath)

		fullPath := filepath.Join(h.Directory, urlPath)
		fi, err := os.Stat(fullPath)
		if err != nil {
			log.Printf("Error getting file info: %v", err)
			fileHandler.ServeHTTP(w, r)
			return
		}

		if fi.IsDir() {
			log.Printf("Directory detected, serving with template: %s", urlPath)

			rec := httptest.NewRecorder()
			fileHandler.ServeHTTP(rec, r)

			tmpl, err := template.ParseFS(h.FS,
				"templates/base.html",
				"templates/filelist.html",
			)
			if err != nil {
				log.Printf("Failed to parse template: %v", err)
				for k, v := range rec.Header() {
					w.Header()[k] = v
				}
				w.WriteHeader(rec.Code)
				_, err := rec.Body.WriteTo(w)
				if err != nil {
					log.Println(err)
				}
				return
			}

			data := struct {
				Title       string
				FileContent template.HTML
				GetFiles    string
				UploadFiles string
				Clipboard   string
				ToQrcode    string
			}{
				Title:       "File Browser",
				FileContent: template.HTML(rec.Body.String()),
				GetFiles:    "/file/",
				UploadFiles: "/upload",
				Clipboard:   "/clipboard",
				ToQrcode:    "/qrcode",
			}

			if h.BaseURI != "" {
				log.Printf("Using full URLs with baseURI: %s", h.BaseURI)
				data.GetFiles = h.BaseURI + "/file/"
				data.UploadFiles = h.BaseURI + "/upload"
				data.Clipboard = h.BaseURI + "/clipboard"
				data.ToQrcode = h.BaseURI + "/qrcode"
			}

			err = tmpl.Execute(w, data)
			if err != nil {
				log.Printf("Failed to execute template: %v", err)
				http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		log.Printf("Serving file: %s", urlPath)
		fileHandler.ServeHTTP(w, r)
	}
}
