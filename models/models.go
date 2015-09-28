package models

import (
	"crypto/tls"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type BasicAuth struct {
	User     string
	Password string
}

type Result struct {
	err          error
	Response     *http.Response
	TotalSeconds time.Duration
}

type RequestOptions struct {
	Method    string
	TargetUrl *url.URL
	Headers   http.Header
	Body      string
	Auth      BasicAuth
}

type LoadTester struct {
	Options     RequestOptions
	NRequests   int
	NConcurrent int
	Timeout     int
	VerifyHttps bool
	ProxyUrl    *url.URL
	Results     chan *Result
	StartTime   time.Time
}

func (r *RequestOptions) Request() *http.Request {
	request, err := http.NewRequest(r.Method, r.TargetUrl.String(), strings.NewReader(r.Body))
	if err != nil {
		log.Fatal("Unable to build request object.")
	}
	request.Header = r.Headers
	if r.Auth.User != "" && r.Auth.Password != "" {
		request.SetBasicAuth(r.Auth.User, r.Auth.Password)
	}
	return request
}

func (l *LoadTester) Run() {
	l.Results = make(chan *Result, l.NRequests)
	l.initiateTest()
	// l.printResults()
	close(l.Results)
}

func (l *LoadTester) initiateTest() {
	var waitGroup sync.WaitGroup
	waitGroup.Add(l.NRequests)

	requestJobs := make(chan *http.Request, l.NRequests)
	completedCounter := make(chan int)
	for i := 0; i < l.NConcurrent; i++ {
		go func() {
			l.requestWorker(&waitGroup, requestJobs, completedCounter)
		}()
	}
	for i := 0; i < l.NRequests; i++ {
		requestJobs <- l.Options.Request()
	}
	go func() {
		divisor := math.Floor(float64(l.NRequests / 10))
		//for completedCount := len(*l.Results); completedCount < l.NRequests; {
		lastPrintedCount := 0
		sum := 0
		for i := range completedCounter {
			sum += i
			if math.Mod(float64(sum), divisor) == 0 {
				if sum > lastPrintedCount {
					requestsPerSecond := float64(sum) / time.Now().Sub(l.StartTime).Seconds()
					fmt.Print("Completed: ", sum)
					fmt.Printf("  Requests Per Second: %.2f\n\n", requestsPerSecond)
					lastPrintedCount = sum
				}
			}
		}
	}()
	close(requestJobs)
	waitGroup.Wait()
}

func (l *LoadTester) requestWorker(waitGroup *sync.WaitGroup, ch chan *http.Request, counter chan int) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: l.VerifyHttps,
		},
		TLSHandshakeTimeout: time.Duration(l.Timeout) * time.Millisecond,
	}
	if l.ProxyUrl.String() != "" {
		transport.Proxy = http.ProxyURL(l.ProxyUrl)
	}
	client := &http.Client{Transport: transport}
	for request := range ch {
		startTime := time.Now()
		response, err := client.Do(request)
		if err != nil {
			log.Println(err)
			log.Fatal("Unable to complete request.")
		}
		waitGroup.Done()
		l.Results <- &Result{
			err,
			response,
			time.Now().Sub(startTime),
		}
		counter <- 1
	}
}

//func (l *LoadTester) printSummaryStats() {
//	totalDuration := time.Now().Sub(l.StartTime)
//	len(range l.Results)
//}
