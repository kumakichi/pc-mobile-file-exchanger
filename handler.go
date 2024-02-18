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
	"time"
)

var (
	autoBan map[string]banip
)

type banip struct {
	count int
	ts    int
}

func init() {
	autoBan = make(map[string]banip, 10)
}

func askForAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Please Login"`)
	w.Header().Set("Content-type", "text/html")
	w.WriteHeader(http.StatusUnauthorized)
}

func procAutoBan(m map[string]banip, r *http.Request) (needBan bool) {
	var requestIP string
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		requestIP = forwarded
	} else {
		requestIP = r.RemoteAddr
	}
	now := time.Now().Second()

	v, ok := m[requestIP]
	if ok && v.count > banCount && v.ts-now < banTimeout {
		needBan = true
		m[requestIP] = banip{
			ts:    now,
			count: v.count + 1,
		}
		log.Printf("=== autoBan:%s,count:%d ===\n", requestIP, m[requestIP].count)
		return
	}

	if !ok {
		m[requestIP] = banip{count: 1}
	} else {
		cnt := v.count
		if v.ts-now > banTimeout {
			cnt = 0
		}
		m[requestIP] = banip{
			ts:    v.ts,
			count: cnt + 1,
		}
	}

	if m[requestIP].count > banCount {
		v = m[requestIP]
		m[requestIP] = banip{
			count: v.count,
			ts:    now,
		}
		log.Printf("=== autoBan add:%s ===\n", requestIP)
		needBan = true
	}

	return
}

func authMiddleware(next http.Handler) http.Handler {
	if noAuth {
		return next
	}
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != authStr && procAutoBan(autoBan, r) {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		var err error
		switch auth {
		case "": // no such header
			askForAuth(rw)
			_, err = rw.Write([]byte("no auth header received"))
		case authStr: // passed
			next.ServeHTTP(rw, r)
		default: // header existed but invalid
			askForAuth(rw)
			_, err = rw.Write([]byte("not authenticated"))
		}
		if err != nil {
			log.Printf("error: %v", err)
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
Clipboard: clipboardPattern,
			},
			ToQrcode: qrPattern,
			Title:    "Upload Files",
			FontSize: getFontSize(r),
		}
		err := t.Execute(w, data)
		if err != nil {
			log.Printf("err: %v", err)
		}
	} else {
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			log.Printf("err: %v", err)
		}

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
				_, err = io.Copy(bf, file)
				if err != nil {
					log.Printf("err:%v", err)
				}
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
				GetFiles:    filePattern,
					UploadFiles: uploadPattern,
Clipboard: clipboardPattern,
				},
				ToQrcode: qrPattern,
				Title:    "Upload Result",
				FontSize: getFontSize(r),
			},
			OkFiles:     okFiles,
			FailedFiles: failedFiles,
			FilePath:    absPath,
		}

		err = t.Execute(w, data)
		if err != nil {
			log.Printf("err: %v", err)
		}
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
Clipboard: clipboardPattern,
				},
				ToQrcode: qrPattern,
				Title:    "Get Files",
				FontSize: getFontSize(r),
			}
			err := t.Execute(w, data)
			if err != nil {
				log.Printf("err: %v", err)
			}
			h.ServeHTTP(w, r)
		} else {
			h.ServeHTTP(w, r)
		}
	}
}
