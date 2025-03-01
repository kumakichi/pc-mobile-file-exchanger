package handlers

import (
	"hash/fnv"
	"html/template"
	"io/fs"
	"log"
	mathRand "math/rand"
	"net/http"
	"sync"
	"time"
)

// ClipboardHandler handles clipboard-related requests
type ClipboardHandler struct {
	FS            fs.FS
	BaseURI       string
	clipboardData map[string]string
	fingerMap     map[uint64]string
	mutex         sync.RWMutex
}

// NewClipboardHandler creates a new ClipboardHandler
func NewClipboardHandler(fs fs.FS, baseURI string) *ClipboardHandler {
	return &ClipboardHandler{
		FS:            fs,
		BaseURI:       baseURI,
		clipboardData: make(map[string]string),
		fingerMap:     make(map[uint64]string),
		mutex:         sync.RWMutex{},
	}
}

// ClipboardIndexHandler serves the clipboard page
func (h *ClipboardHandler) ClipboardIndexHandler(w http.ResponseWriter, _ *http.Request) {
	tmpl, err := template.ParseFS(
		h.FS,
		"templates/base.html",
		"templates/clipboard.html",
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
		Title:       "Online Clipboard",
		GetFiles:    "/file/",
		UploadFiles: "/upload",
		Clipboard:   "/clipboard",
		ToQrcode:    "/qrcode",
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
	}
}

// GenerateClipboardCode generates a unique code for clipboard content
func (h *ClipboardHandler) GenerateClipboardCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Content cannot be empty", http.StatusBadRequest)
		return
	}

	finger := fingerprint([]byte(content))
	h.mutex.RLock()
	code, ok := h.fingerMap[finger]
	h.mutex.RUnlock()
	if !ok {
		for i := 0; i < 10; i++ {
			code = generateUniqueCode()
			h.mutex.RLock()
			_, ok := h.clipboardData[code]
			h.mutex.RUnlock()
			if !ok {
				break
			}
		}

		// Store content with code and finger
		h.mutex.Lock()
		h.clipboardData[code] = content
		h.fingerMap[finger] = code
		h.mutex.Unlock()
	}

	w.Header().Set("Content-Type", "text/plain")
	_, err = w.Write([]byte(code))
	if err != nil {
		log.Println(err)
	}
}

func fingerprint(b []byte) uint64 {
	hash := fnv.New64a()
	hash.Write(b)
	return hash.Sum64()
}

func generateUniqueCode() string {
	const charset = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLength := 3
	b := make([]byte, codeLength)
	r := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

// RetrieveClipboardContent retrieves content by code
func (h *ClipboardHandler) RetrieveClipboardContent(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code parameter is required", http.StatusBadRequest)
		return
	}

	h.mutex.RLock()
	content, exists := h.clipboardData[code]
	h.mutex.RUnlock()

	if !exists {
		http.Error(w, "Invalid code or content not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte(content))
	if err != nil {
		log.Println(err)
	}
}
