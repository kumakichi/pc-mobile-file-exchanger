package main

import (
	"flag"
	"fmt"
	"github.com/skip2/go-qrcode"
	"github.com/skratchdot/open-golang/open"
	"log"
	"net/http"
	"time"
)

var (
	port      int
	help      bool
	directory string
	timeout   int
)

const (
	qrPattern   = "/qrcode"
	filePattern = "/file/"
)

func init() {
	flag.IntVar(&port, "p", 8000, "Listen port")
	flag.BoolVar(&help, "h", false, "Print this help infomation")
	flag.StringVar(&directory, "d", ".", "File server root path")
	flag.IntVar(&timeout, "t", 5, "Select timeout in seconds, when you have more than 1 NIC, you need to select one, or we will use all the NICs")
}

func main() {
	flag.Parse()
	if help {
		flag.PrintDefaults()
		return
	}

	ips := getIPs()
	fmt.Println(selectInterface(ips))
	return

	http.HandleFunc(qrPattern, func(w http.ResponseWriter, r *http.Request) {
		b, err := qrcode.Encode("http://192.168.1.5:8000/file", qrcode.Highest, 256)
		if err != nil {
			log.Fatal(err)
		} else {
			w.Write(b)
		}
	})
	http.Handle(filePattern, http.StripPrefix(filePattern, http.FileServer(http.Dir(directory))))

	host := fmt.Sprintf(":%d", port)
	log.Printf("Listen at %s\n", host)
	open.Run("http://localhost" + host + "/qrcode")
	log.Fatal(http.ListenAndServe(host, nil))
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
		//TODO: user select
		//return ""
		select {
		case <-time.After(time.Second * time.Duration(timeout)):
			fmt.Println("user not inputted")
			return ""
		case input, ok := <-ch:
			if ok && input >= 0 && input < len(keys) {
				fmt.Printf("Using %s\t%s\n", keys[input], ips[keys[input]])
			} else {
				fmt.Printf("Invalid index got, we will listen on all interfaces.\n")
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

	fmt.Printf("Please input the interface index: ")
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
