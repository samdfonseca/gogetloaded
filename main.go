package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"

	"github.com/samdfonseca/gogetloaded/models"
	"time"
)

var (
	method            = flag.String("m", "GET", "")
	headers           = flag.String("h", "", "")
	contentType       = flag.String("T", "application/json", "")
	body              = flag.String("b", "", "")
	basicAuthUser     = flag.String("u", "", "")
	basicAuthPassword = flag.String("p", "", "")
	proxy             = flag.String("proxy", "", "")
	verifyHttps       = flag.Bool("verify", false, "")
	nRequests         = flag.Int("n", 100, "")
	nConcurrent       = flag.Int("c", 20, "")
	timeout           = flag.Int("t", 0, "")
	cpus              = flag.Int("C", runtime.GOMAXPROCS(-1), "")
)

type LoadTestConfig struct {
	Id       string
	Headers  string
	Url      string
	Method   string
	DataMode string
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("No arguments given. Exiting...")
	}

	runtime.GOMAXPROCS(*cpus)

	if *nRequests < 1 || *nConcurrent < 1 {
		log.Fatal("Number of requests/concurrency can not be less than 1.")
	}
	if *nConcurrent > *nRequests {
		log.Fatal("Number of requests can not be greater than concurrency.")
	}

	if *method != "GET" && *method != "POST" && *method != "PUT" {
		log.Fatal("Method must be GET, POST or PUT.")
	}

	fmt.Println("Starting Load Test...")
	var (
		targetUrl, tUrlErr = url.Parse(flag.Args()[0])
		header             = make(http.Header)
	)

	if tUrlErr != nil {
		log.Fatal("Unable to parse target URL.")
	}

	header.Set("Content-Type", *contentType)
	if *headers != "" {
		for _, h := range strings.Split(*headers, ";") {
			re := regexp.MustCompile("^([\\w-]+):\\s*(.+)")
			match := re.FindStringSubmatch(h)
			if len(match) >= 1 {
				header.Set(match[1], match[2])
			}
		}
	}

	proxyUrl := new(url.URL)
	if pUrl, proxyUrlErr := url.Parse(*proxy); proxyUrlErr != nil {
		log.Fatal("Unable to parse proxy URL.")
	} else if pUrl.String() != "" {
		proxyUrl = pUrl
	}

	requestBody := *body
	if strings.Index(*body, "@") == 0 {
		contents, err := ioutil.ReadFile(*body)
		if err != nil {
			log.Fatal("Unable to read request data from file.")
		}
		requestBody = string(contents)
	}

	(&models.LoadTester{
		models.RequestOptions{
			*method,
			targetUrl,
			header,
			requestBody,
			models.BasicAuth{
				*basicAuthUser,
				*basicAuthPassword,
			},
		},
		*nRequests,
		*nConcurrent,
		*timeout,
		*verifyHttps,
		proxyUrl,
		nil,
		time.Now(),
	}).Run()
}
