package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kumakichi/pc-mobile-file-exchanger/internal/auth"
	fsInternal "github.com/kumakichi/pc-mobile-file-exchanger/internal/fs"
	"github.com/kumakichi/pc-mobile-file-exchanger/internal/handlers"
	"github.com/kumakichi/pc-mobile-file-exchanger/internal/utils"
	"github.com/skratchdot/open-golang/open"
)

//go:embed static/webfonts/fa-solid-900.woff2
var webfont []byte

//go:embed static/css/font-awesome_5.15.4_all.min.css
var stylesCSS2 []byte

//go:embed static/css/styles.css
var stylesCSS []byte

//go:embed templates/*.html
var templateFs embed.FS

const (
	qrPattern        = "/qrcode"
	filePattern      = "/file/"
	uploadPattern    = "/upload"
	clipboardPattern = "/clipboard"
)

var (
	Version           = "unknown"
	directory         string
	upDirectory       string
	port              int
	help              bool
	noAuth            bool
	noQRCode          bool
	patchHTMLToParent bool
	baseURI           string
	filterSuffix      string
	authUser          string
	authPwd           string
	banTimeoutVar     int
	banCountVar       int
	serverKey         string
	serverCrt         string
	netInterfaceIndex int
)

func init() {
	flag.IntVar(&port, "port", 8000, "the listening port")
	flag.BoolVar(&help, "h", false, "show this help message")
	flag.StringVar(&directory, "d", "./", "directory for sharing")
	flag.StringVar(&upDirectory, "ud", "./", "directory for uploading files")
	flag.BoolVar(&noAuth, "na", false, "no authentication")
	flag.BoolVar(&noQRCode, "nq", false, "no QRCode page")
	flag.BoolVar(&patchHTMLToParent, "pp", false, "patch html file with parent links")
	flag.StringVar(&filterSuffix, "fs", "", "filter by suffix, empty means do not filter")
	flag.StringVar(&authUser, "au", "admin", "username for basic auth")
	flag.StringVar(&authPwd, "ap", "admin", "password for basic auth")
	flag.IntVar(&banTimeoutVar, "banTimeout", 300, "timeout for auto ban")
	flag.IntVar(&banCountVar, "banCount", 3, "count for auto ban")
	flag.StringVar(&serverKey, "key", "", "server key")
	flag.StringVar(&serverCrt, "crt", "", "server cert")
	flag.IntVar(&netInterfaceIndex, "nic", -1, "network interface index, use -1 to choose interactively")
}

func main() {
	flag.Parse()
	if help {
		fmt.Printf("Usage: %s [options]\n", os.Args[0])
		fmt.Printf("Version: %s\n\n", Version)
		flag.PrintDefaults()
		return
	}

	var authString string
	if !noAuth {
		authString = auth.CalcAuthStr(authUser, authPwd)
	}

	ips := utils.GetIPs()
	var ip string
	if netInterfaceIndex >= 0 {
		keys := utils.Keys(ips)
		if netInterfaceIndex < len(keys) {
			ip = ips[keys[netInterfaceIndex]]
		} else {
			log.Fatal("Invalid network interface index.")
		}
	} else {
		ip = selectInterface(ips)
	}
	host := fmt.Sprintf("%s:%d", ip, port)
	baseURI = "http://" + host

	// Initialize file system
	fileSystem := fsInternal.CreateFilesystemHandler(directory, filterSuffix)

	// Initialize handlers
	fileHandlerObj := handlers.NewFileHandler(templateFs, baseURI, directory, filterSuffix, patchHTMLToParent)
	uploadHandler := handlers.NewUploadHandler(templateFs, baseURI, upDirectory)
	clipboardHandler := handlers.NewClipboardHandler(templateFs, baseURI)
	qrcodeHandler := handlers.NewQRCodeHandler(templateFs, baseURI, qrPattern)

	// Set up routes
	// Serve static files with proper MIME types
	http.HandleFunc("/static/css/styles.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, err := w.Write(stylesCSS)
		if err != nil {
			log.Println(err)
		}
	})
	http.HandleFunc("/static/css/font-awesome_5.15.4_all.min.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, err := w.Write(stylesCSS2)
		if err != nil {
			log.Println(err)
		}
	})
	http.HandleFunc("/static/webfonts/fa-solid-900.woff2", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream; charset=utf-8")
		_, err := w.Write(webfont)
		if err != nil {
			log.Println(err)
		}
	})

	http.Handle(qrPattern, auth.Middleware(
		http.HandlerFunc(qrcodeHandler.QRCodeHandler), authString, noAuth, banTimeoutVar, banCountVar))

	// 恢复原来的文件处理程序注册
	http.Handle(filePattern, auth.Middleware(
		http.StripPrefix(filePattern, fileHandlerObj.WrapFSHandler(http.FileServer(http.FS(fileSystem)))),
		authString, noAuth, banTimeoutVar, banCountVar))

	http.Handle(uploadPattern, auth.Middleware(
		http.HandlerFunc(uploadHandler.HandleUpload),
		authString, noAuth, banTimeoutVar, banCountVar))
	http.Handle(clipboardPattern, auth.Middleware(
		http.HandlerFunc(clipboardHandler.ClipboardIndexHandler),
		authString, noAuth, banTimeoutVar, banCountVar))

	http.Handle(clipboardPattern+"/generate", auth.Middleware(
		http.HandlerFunc(clipboardHandler.GenerateClipboardCode),
		authString, noAuth, banTimeoutVar, banCountVar))

	http.Handle(clipboardPattern+"/retrieve", auth.Middleware(
		http.HandlerFunc(clipboardHandler.RetrieveClipboardContent),
		authString, noAuth, banTimeoutVar, banCountVar))

	// Start server
	log.Printf("Listen at %s\n", host)
	log.Printf("Access files by http://%s%s\n", host, filePattern)

	if !noQRCode {
		err := open.Run(baseURI + qrPattern)
		if err != nil {
			log.Printf("err: %v", err)
		}
	}

	if serverKey == "" || serverCrt == "" {
		log.Fatal(http.ListenAndServe(host, nil))
	} else {
		log.Fatal(http.ListenAndServeTLS(host, serverCrt, serverKey, nil))
	}
}

func selectInterface(ips map[string]string) string {
	length := len(ips)
	ch := make(chan int, 1)

	switch {
	case length < 1:
		log.Println("Can not get local ip")
		return "127.0.0.1"
	case length == 1:
		for _, v := range ips {
			return v
		}
	default:
		keys := utils.Keys(ips)
		go readUserInput(keys, ips, ch)
		select {
		case <-time.After(time.Second * 30):
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

func readUserInput(keys []string, ips map[string]string, ch chan int) {
	for i, k := range keys {
		fmt.Printf("(%d): %s\t%s\n", i, k, ips[k])
	}
	fmt.Printf("Select interface by index [0-%d] ? ", len(keys)-1)

	var i int
	_, err := fmt.Scanf("%d", &i)
	if err != nil {
		log.Fatal(err)
	}
	ch <- i
}
