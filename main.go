package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/skratchdot/open-golang/open"
)

var (
	port              int
	help              bool
	directory         string
	upDirectory       string
	timeout           int
	noQRCode          bool
	noDir             bool
	patchHtmlToParent bool
	baseURI           string
	filterSuffix      string
	filenameContains  string
	qrBase64          string
	authStr           string
	authUser          string
	authPwd           string
	noAuth            bool
	banTimeout        int
	banCount          int
	serverKey         string
	serverCrt         string
)

type GetOrUpload struct {
	GetFiles    string
	UploadFiles string
}

type QrPage struct {
	QrBase string
}

type Head struct {
	GetOrUpload
	ToQrcode string
	Title    string
	FontSize int
	QrPage
}

type UpResult struct {
	Head
	OkFiles     string
	FailedFiles string
	FilePath    string
}

const (
	qrPattern     = "/qrcode"
	filePattern   = "/file/"
	uploadPattern = "/upload"
	patchHtmlName = "pp"
)

func init() {
	flag.IntVar(&port, "p", 8000, "Listen port")
	flag.BoolVar(&help, "h", false, "Print this help infomation")
	flag.StringVar(&directory, "d", ".", "File server root path")
	flag.StringVar(&upDirectory, "u", ".", "Upload files root path")
	flag.StringVar(&filterSuffix, "s", "", "Suffix of filename, only matched file will be shown")
	flag.StringVar(&filenameContains, "f", "", "Substring of filename, only matched file/dirs will be shown")
	flag.IntVar(&timeout, "t", 5, "Select timeout in seconds, when you have more than 1 NIC, you need to select one, or we will use all the NICs")
	flag.BoolVar(&noQRCode, "n", false, "Do not open browser automatically")
	flag.BoolVar(&noDir, "nd", false, "Do not show directory, serve only files")
	flag.BoolVar(&noAuth, "na", false, "Do not auth")
	flag.StringVar(&authUser, "au", "share", "auth user")
	flag.StringVar(&authPwd, "ap", "share", "auth pwd")
	flag.IntVar(&banTimeout, "bt", 3600, "auto ban timeout")
	flag.IntVar(&banCount, "bc", 5, "max fail count before auto ban")
	flag.BoolVar(&patchHtmlToParent, patchHtmlName, false, "patch html, add link to parent")
	flag.StringVar(&serverKey, "sk", "", "tls: server key")
	flag.StringVar(&serverCrt, "sc", "", "tls: server secret")
}

func qrServePage(w http.ResponseWriter, r *http.Request) {
	if qrBase64 == "" {
		b, err := qrcode.Encode(baseURI+filePattern, qrcode.Highest, 256)
		if err != nil {
			log.Fatal(err)
		}
		qrBase64 = base64.StdEncoding.EncodeToString(b)
	}

	t, err := template.New("qrcode").Parse(xxqrTemplate)
	if err != nil {
		log.Fatal(err)
	}

	data := Head{
		GetOrUpload: GetOrUpload{
			GetFiles:    filePattern,
			UploadFiles: uploadPattern,
		},
		ToQrcode: qrPattern,
		Title:    "Index Page",
		FontSize: getFontSize(r),
		QrPage: QrPage{
			QrBase: qrBase64,
		},
	}
	err = t.Execute(w, data)
	if err != nil {
		log.Printf("err: %v", err)
	}
}

func main() {
	flag.Parse()
	if help {
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
		return
	}

	if !noAuth {
		authStr = calcAuthStr(authUser, authPwd)
	}

	ips := getIPs()
	ip := selectInterface(ips)
	host := fmt.Sprintf("%s:%d", ip, port)
	baseURI = "http://" + host

	http.Handle(qrPattern, authMiddleware(http.HandlerFunc(qrServePage)))
	http.Handle(filePattern, authMiddleware(http.StripPrefix(filePattern, wrapFSHandler(http.FileServer(http.FS(suffixDirFS(directory)))))))
	http.Handle(uploadPattern, authMiddleware(http.HandlerFunc(uploadHandler)))

	log.Printf("Listen at %s\n", host)
	log.Printf("Access files by http://%s\n", host+filePattern)

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
		keys := keys(ips)
		go readUserInput(keys, ips, ch)
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

func readUserInput(keys []string, ips map[string]string, ch chan int) {
	defer func() { close(ch) }()
	fmt.Printf("You hava more than 1 NIC, please select one, or we listen on all the NICs.\n\n")
	for i, v := range keys {
		fmt.Printf("%2d\t%-16s\t%-s\n", i, v, ips[v])
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
	sort.Strings(keys)
	return keys
}

func getFontSize(r *http.Request) int {
	if isMobile(r.Header.Get("User-Agent")) {
		return 300
	}

	return 100
}

func isMobile(ua string) bool {
	ua = strings.ToLower(ua)

	mobileUAs := map[string]struct{}{
		"iphone":        {},
		"ipod":          {},
		"ipad":          {},
		"android":       {},
		"windows phone": {},
		"googletv":      {},
	}
	for k := range mobileUAs {
		if strings.Contains(ua, k) {
			return true
		}
	}

	return false
}
