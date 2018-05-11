package main

import (
	"flag"
	"fmt"
	"github.com/skip2/go-qrcode"
	"github.com/skratchdot/open-golang/open"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	port        int
	help        bool
	directory   string
	upDirectory string
	timeout     int
	noqrcode    bool
	baseURI     string
)

type FromTo struct {
	FromPC string
	ToPC   string
}

type UpResult struct {
	FromTo
	ToIndex  string
	FileName string
	FilePath string
}

const (
	qrPattern     = "/qrcode"
	filePattern   = "/file/"
	indexPattern  = "/index"
	uploadPattern = "/upload"
)

func init() {
	flag.IntVar(&port, "p", 8000, "Listen port")
	flag.BoolVar(&help, "h", false, "Print this help infomation")
	flag.StringVar(&directory, "d", ".", "File server root path")
	flag.StringVar(&upDirectory, "u", ".", "Upload files root path")
	flag.IntVar(&timeout, "t", 5, "Select timeout in seconds, when you have more than 1 NIC, you need to select one, or we will use all the NICs")
	flag.BoolVar(&noqrcode, "n", false, "Only serve file, do not generate and open QR code")
}

func main() {
	flag.Parse()
	if help {
		flag.PrintDefaults()
		return
	}

	ips := getIPs()
	ip := selectInterface(ips)
	host := fmt.Sprintf("%s:%d", ip, port)
	baseURI = "http://" + host

	if noqrcode == false {
		http.HandleFunc(qrPattern, func(w http.ResponseWriter, r *http.Request) {
			b, err := qrcode.Encode(baseURI+indexPattern, qrcode.Highest, 256)
			if err != nil {
				log.Fatal(err)
			} else {
				w.Write(b)
			}
		})
	}
	http.Handle(filePattern, http.StripPrefix(filePattern, wrapHandler(http.FileServer(http.Dir(directory)))))
	http.HandleFunc(uploadPattern, uploadHandler)
	http.HandleFunc(indexPattern, indexHandler)

	log.Printf("Listen at %s\n", host)
	log.Printf("Access files by http://%s\n", host+filePattern)

	if noqrcode == false {
		open.Run(baseURI + qrPattern)
	}
	log.Fatal(http.ListenAndServe(host, nil))
}

func wrapHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		style := `
<!DOCTYPE html>
<head>
    <title>Choose</title>
    <style>
        pre {
            text-align: center;
            font-size:300%;
            margin: auto;
        }
    </style>
</head>
<body>
`
		w.Write([]byte(style))
		h.ServeHTTP(w, r)
		w.Write([]byte("</body>"))
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.New("index").Parse(indexTemplate)
	t.Execute(w, FromTo{baseURI + filePattern, baseURI + uploadPattern})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.New("up").Parse(uploadTemplate)
		t.Execute(w, nil)
	} else {
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("upfile")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()
		upFilePath := filepath.Join(upDirectory, handler.Filename)
		f, err := os.OpenFile(upFilePath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(err)
			fmt.Fprintf(w, "%v", err)
			return
		}
		defer f.Close()
		io.Copy(f, file)
		absPath, _ := filepath.Abs(upDirectory)
		t, _ := template.New("result").Parse(upResultTemplate)
		t.Execute(w, UpResult{FromTo{baseURI + filePattern, baseURI + uploadPattern}, baseURI + indexPattern, handler.Filename, absPath})
	}
}

func selectInterface(ips map[string]string) string {
	length := len(ips)
	ch := make(chan int, 1)
	defer func() { close(ch) }()

	switch {
	case length < 1:
		log.Fatal("Can not get local ip")
	case length == 1:
		for _, v := range ips {
			return v
		}
	default:
		keys := keys(ips)
		go readUserInput(keys, ch)
		select {
		case <-time.After(time.Second * time.Duration(timeout)):
			fmt.Println()
			log.Printf("Input timeout, using %s\t%s\n", keys[0], ips[keys[0]])
			return ips[keys[0]]
		case input, ok := <-ch:
			if ok && input >= 0 && input < len(keys) {
				fmt.Printf("Using %s\t%s\n", keys[input], ips[keys[input]])
				return ips[keys[input]]
			} else {
				log.Fatal("Invalid index.")
			}
		}
	}

	return ""
}

func readUserInput(keys []string, ch chan int) {
	fmt.Printf("You hava more than 1 NIC, please select one, or we listen on all the NICs.\n\n")
	for i, v := range keys {
		fmt.Printf("%d\t(%s)\n", i, v)
	}

	fmt.Printf("Please input the interface index[0]: ")
	var idx int
	fmt.Scanf("%d", &idx)
	ch <- idx
}

func keys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
