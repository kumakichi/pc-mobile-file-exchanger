package main

import (
	"bufio"
	"encoding/base64"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func askForAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Please Login"`)
	w.Header().Set("Content-type", "text/html")
	w.WriteHeader(http.StatusUnauthorized)
}

func authMiddleware(next http.Handler) http.Handler {
	if noAuth {
		return next
	}
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		switch auth {
		case "": // no such header
			askForAuth(rw)
			rw.Write([]byte("no auth header received"))
		case authStr: // passed
			next.ServeHTTP(rw, r)
		default: // header existed but invalid
			askForAuth(rw)
			rw.Write([]byte("not authenticated"))
		}
	})
}

func calcAuthStr(usr, pwd string) string {
	msg := []byte(usr + ":" + pwd)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(encoded, msg)
	return "Basic " + string(encoded)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.New("up").Parse(uploadTemplate)
		data := Head{
			GetOrUpload: GetOrUpload{
				GetFiles:    filePattern,
				UploadFiles: uploadPattern,
			},
			ToQrcode: qrPattern,
			Title:    "Upload Files",
			FontSize: getFontSize(r),
		}
		t.Execute(w, data)
	} else {
		r.ParseMultipartForm(32 << 20)

		type upStat struct {
			name string
			isOk bool
		}
		okFiles := ""
		failedFiles := ""
		upStatCh := make(chan upStat)
		go func() {
			for {
				if v, ok := <-upStatCh; ok {
					if v.isOk {
						okFiles += "," + v.name
					} else {
						failedFiles += "," + v.name
					}
				} else {
					break
				}
			}
		}()

		fhs := r.MultipartForm.File["uploadFile"]
		wg := &sync.WaitGroup{}
		wg.Add(len(fhs))
		taskCh := make(chan struct{}, 5)
		for i := range fhs {
			func(i int) {
				taskCh <- struct{}{}
				fh := fhs[i]
				defer wg.Done()
				defer func() { <-taskCh }()

				file, err := fh.Open()
				// f is one of the files
				if err != nil {
					log.Println(err)
					return
				}
				defer file.Close()
				fname := fh.Filename
				if fname == "" {
					return // not selected
				}

				upFilePath := filepath.Join(upDirectory, fname)
				f, err := os.OpenFile(upFilePath, os.O_WRONLY|os.O_CREATE, 0666)
				bf := bufio.NewWriter(f)
				if err != nil {
					upStatCh <- upStat{fname, false}
					log.Println(err)
					return
				}
				defer f.Close()
				io.Copy(bf, file)
				upStatCh <- upStat{fname, true}
			}(i)
		}
		wg.Wait()

		close(upStatCh)

		absPath, _ := filepath.Abs(upDirectory)
		t, _ := template.New("result").Parse(upResultTemplate)
		okFiles = strings.Replace(okFiles, ",", "", 1)
		failedFiles = strings.Replace(failedFiles, ",", "", 1)

		data := UpResult{
			Head: Head{
				GetOrUpload: GetOrUpload{
					UploadFiles: uploadPattern,
				},
				ToQrcode: qrPattern,
				Title:    "Upload Result",
				FontSize: getFontSize(r),
			},
			OkFiles:     okFiles,
			FailedFiles: failedFiles,
			FilePath:    absPath,
		}

		t.Execute(w, data)
	}
}

func wrapFSHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := []rune(r.RequestURI)
		if s[len(s)-1] == '/' {
			t, _ := template.New("head").Parse(customHead)
			data := Head{
				GetOrUpload: GetOrUpload{
					GetFiles:    filePattern,
					UploadFiles: uploadPattern,
				},
				ToQrcode: qrPattern,
				Title:    "Get Files",
				FontSize: getFontSize(r),
			}
			t.Execute(w, data)
			h.ServeHTTP(w, r)
		} else {
			h.ServeHTTP(w, r)
		}
	}
}
