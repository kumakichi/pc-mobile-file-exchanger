package main

import (
	"flag"
	"net/http"
	"log"
	"fmt"
	"github.com/skip2/go-qrcode"
	"github.com/skratchdot/open-golang/open"
)

var (
	port int
	help bool
	directory string
)

func init() {
	flag.IntVar(&port, "p", 8000, "Listen port")
	flag.BoolVar(&help, "h",  false, "Print this help infomation")
	flag.StringVar(&directory, "d", ".", "File server root path")
}

func main() {
	flag.Parse()
	if help {
		flag.PrintDefaults()
		return
	}

	http.HandleFunc("/qrcode", func(w http.ResponseWriter, r *http.Request) {
		b, err :=qrcode.Encode("http://192.168.1.5:8000/file",  qrcode.Highest, 256)
		if err !=nil {
			log.Fatal(err)
		} else {
			w.Write(b)
		}
	})
	http.Handle("/file", http.FileServer(http.Dir(directory)))

	host:=fmt.Sprintf("localhost:%d", port)
	log.Printf("Listen at %s\n", host)
	open.Run("http://"+host+"/qrcode")
	log.Fatal(http.ListenAndServe(host, nil))
}