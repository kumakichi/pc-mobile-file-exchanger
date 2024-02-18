package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/skratchdot/open-golang/open"
)

var (
	Version           string = "unknown"
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
	clipboardData     map[string]string
)

type GetOrUpload struct {
	GetFiles    string
	UploadFiles string
	Clipboard       string
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
	clipboardPattern  = "/clipboard"
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

	clipboardData = make(map[string]string)
	rand.Seed(time.Now().UnixNano())
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
			Clipboard:       clipboardPattern,
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
		fmt.Printf("Usage: %s [options]\n", os.Args[0])
		fmt.Printf("%s\n\n", Version)
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

	http.Handle(clipboardPattern, authMiddleware(http.HandlerFunc(clipboardIndexHandler)))
	http.Handle(clipboardPattern+"/generate", authMiddleware(http.HandlerFunc(generateCodeHandler)))
	http.Handle(clipboardPattern+"/retrieve", authMiddleware(http.HandlerFunc(retrieveHandler)))

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

func clipboardIndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("clipboard").Parse(`
<div class="nav">
  <li><a href="{{ .GetFiles }}" class="child">Get Files</a></li>

  <li><a href="{{ .ToQrcode }}" class="child">QR Code</a></li>

  <li><a href="{{ .UploadFiles }}" class="child">Upload</a></li>

  <li><a href="{{ .Clipboard }}" class="child">Clipboard</a></li>

  <li><a href="../" class="child">../</a></li>
</div>
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Online Clipboard</title>
		<style>
			body {
				font-family: 'Arial', sans-serif;
				text-align: center;
				margin: 20px;
			}
			h1 {
				color: #333;
			}
			form {
				margin-top: 20px;
			}
			label {
				display: block;
				margin-bottom: 5px;
				color: #666;
			}
			textarea {
				width: 90%;
				padding: 10px;
				box-sizing: border-box;
			}
			.code-input {
    padding: 2px 2px;
    border-radius: 3px;
    box-shadow: 0px 2px 6px rgba(19, 18, 66, 0.07);
    border-width: 2px;
				width: 30%;
			}
			button {
				padding: 10px;
				background-color: #4CAF50;
				color: white;
				border: none;
				border-radius: 4px;
				cursor: pointer;
			}

		.copypaste {
		    border-radius: 13px;
		    padding: 15px;
		    padding-bottom: 32px;
		    border: solid 2px #f9cd25;
		    width: 90%!important;
		}

    #content {
        display: inline-block;
        width: 74%; /* 90% - 16% = 74% */
        height: 60%;
        box-sizing: border-box;
    }

    #generateButton {
        display: inline-block;
        width: 24%; /* 90% - 16% = 74% + 2% margin */
        margin-left: 2%; /* 2% margin */
    }

.row {
    display: flex;
    flex-wrap: wrap;
}

.row-2 {
    flex: 0 0 auto;
    width: 20%;
}

.button-container {
    margin-top: 10px;
    flex: 0 0 auto;
    width: 80%;
    margin-left: auto; /* 将 button-container 靠右对齐 */
}

		</style>
		<script>
			function generateCode() {
			    const host = window.location.hostname;
			    const port = window.location.port;
				const content = document.getElementById("content").value;

				fetch(`+"`"+"http://${host}:${port}/clipboard/generate"+"`"+`, {
					method: "POST",
					headers: {
						"Content-Type": "application/x-www-form-urlencoded",
					},
					body: "content=" + encodeURIComponent(content),
				})
					.then(response => response.text())
					.then(code => {
						document.getElementById("code").value = code;
					})
					.catch(error => console.error('Error generating code:', error));
			}

			function retrieveContent() {
			    const host = window.location.hostname;
			    const port = window.location.port;
				const code = document.getElementById("code").value;
				fetch(`+"`"+"http://${host}:${port}/clipboard/retrieve?code="+"`"+` + code)
					.then(response => response.text())
					.then(content => {
						document.getElementById("content").value = content;
					})
					.catch(error => console.error('Error retrieving content:', error));
			}
		</script>
	</head>
	<body>
		<h1>Online Clipboard</h1>
    <form action="/clipboard/generate" method="post" id="clipboardForm">
            <textarea id="content" name="content" class="copypaste" placeholder="Text Goes here"></textarea>
        <div class="row">
        <div class="row-2"></div>
        <div class="button-container">
            <button type="button" id="generateButton" onclick="generateCode()">Share</button>
			<input type="text" id="code" name="code" class="code-input" required>
			<button type="button" onclick="retrieveContent()">Retrieve</button>
        </div>
        </div>
    </form>


	</body>
	</html>
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
		data := Head{
			GetOrUpload: GetOrUpload{
				GetFiles:    filePattern,
				UploadFiles: uploadPattern,
Clipboard: clipboardPattern,
			},
			ToQrcode: qrPattern,
			Title:    "Online Clipboard",
			FontSize: getFontSize(r),
		}
	tmpl.Execute(w, data)
}

func generateCodeHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	content := r.Form.Get("content")
	if content == "" {
		http.Error(w, "Content cannot be empty", http.StatusBadRequest)
		return
	}

	code := generateUniqueCode()
	clipboardData[code] = content

	w.Write([]byte(code))
}

func retrieveHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	content, ok := clipboardData[code]
	if !ok {
		http.Error(w, "Invalid code", http.StatusNotFound)
		return
	}

	w.Write([]byte(content))
}

func generateUniqueCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	codeLength := 6
	b := make([]byte, codeLength)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
