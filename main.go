// Crawl a website, breadth-first, listing all unique paths in & out of scope + files + emails.
package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type paths struct {
	url *url.URL
}

var foundExtURL,
    uniqueExtURL,
    emailMatches,
    linksToFilesInScope,
    linksToImagesInScope,
    linksToFilesOutScope,
    linksToImagesOutScope []string

var (
	fileExtensions = []string{
		"doc", "docx", "pdf", "txt", "asp", "aspx", "key",
		"odp", "ods", "pps", "ppt", "pptx", "json", "csv",
		"xlr", "xls", "xlsx", "dat", "db", "dbf", "log",
		"mdb", "sav", "sql", "xml", "zip", "gz", "tar",
		"jar", "7z", "arj", "deb", "pkg", "rar", "rpm",
		"z", "bin", "dmg", "iso", "toast", "vcd", "apk",
		"ipa", "bat", "cgi", "pl", "exe", "py", "wsf",
		"bak", "cab", "cfg", "cpl", "cur", "dll", "dmp",
		"drv", "ini", "msi", "sys", "tmp", "odt", "rtf",
		"tex", "wks", "wps", "wpd", "c", "cpp", "cs", "h",
		"java", "sh", "swift", "vb", "rss", "js", "jsp",
		"php", "cfm", "cer", "crt", "crl", "pem", "pfx",
		"p12", "csr", "p7b", "p7r", "spc", "der",
	}
	imageExtensions = []string{
		"jpg", "jpeg", "jpe", "jif", "jfif", "jfi", "png",
		"gif", "webp", "tiff", "tif", "psd", "raw", "arw",
		"cr2", "nrw", "k25", "bmp", "dib", "heif", "heic",
		"ind", "indd", "indt", "jp2", "j2k", "jpf", "jpx",
		"jpm", "mj2", "svg", "svgz", "ai", "eps",
	}
	foundPaths  []paths
	startingURL *url.URL
	timeout     time.Duration
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("\n[Simple Breadth-First Web Crawler]")
		fmt.Println("\nUsage:   " + os.Args[0] + " <URL>" + " <timeout(secs)>")
		fmt.Println("\nExample: " + os.Args[0] + " https://www.example.com" + " 10\n")
		os.Exit(1)
	}
	foundExtURL = make([]string, 0)
	t, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal("Error parsing timeout. ", err)
	}
	timeout = time.Duration(time.Duration(t) * time.Second)

	startingURL, err = url.Parse(os.Args[1])
	if err != nil {
		log.Fatal("Error parsing starting URL. ", err)
	}
	fmt.Printf("\n%s%s%s\n", "Starting Simple Breadth-First Web Crawling for: [", startingURL.String(), "]")

	crawlHREFInScope(startingURL.Path)

	if foundPaths != nil {
		fmt.Printf("\n%s%d%s\n", "Local paths found: [", len(foundPaths), "]")
		fmt.Printf("%s\n", strings.Repeat("-", 17))
		for _, path := range foundPaths {
			fmt.Printf("%s\n", path.url.String())
		}
	}

	if uniqueExtURL != nil {
		fmt.Printf("\n%s%d%s\n", "External paths found: [", len(uniqueExtURL), "]")
		fmt.Printf("%s\n", strings.Repeat("-", 20))
		for _, uniqExtURL := range uniqueExtURL {
			fmt.Printf("%s\n", uniqExtURL)
		}
	}

	if linksToFilesInScope != nil {
		fmt.Printf("\n%s%d%s\n", "Files in scope found: [", len(linksToFilesInScope), "]")
		fmt.Printf("%s\n", strings.Repeat("-", 20))
		for _, file := range linksToFilesInScope {
			fmt.Printf("%s\n", file)
		}
	} else {
		fmt.Printf("\n%s%d%s\n", "Files in scope found: [", len(linksToFilesInScope), "]")
	}

	if linksToImagesInScope != nil {
		fmt.Printf("\n%s%d%s\n", "Images in scope found: [", len(linksToImagesInScope), "]")
		fmt.Printf("%s\n", strings.Repeat("-", 21))
		for _, image := range linksToImagesInScope {
			fmt.Printf("%s\n", image)
		}
	} else {
		fmt.Printf("\n%s%d%s\n", "Images in scope found: [", len(linksToImagesInScope), "]")
	}

	if linksToFilesOutScope != nil {
		fmt.Printf("\n%s%d%s\n", "Files out scope found: [", len(linksToFilesOutScope), "]")
		fmt.Printf("%s\n", strings.Repeat("-", 21))
		for _, file := range linksToFilesOutScope {
			fmt.Printf("%s\n", file)
		}
	} else {
		fmt.Printf("\n%s%d%s\n", "Files out scope found: [", len(linksToFilesOutScope), "]")
	}

	if linksToImagesOutScope != nil {
		fmt.Printf("\n%s%d%s\n", "Images out scope found: [", len(linksToImagesOutScope), "]")
		fmt.Printf("%s\n", strings.Repeat("-", 21))
		for _, image := range linksToImagesOutScope {
			fmt.Printf("%s\n", image)
		}
	} else {
		fmt.Printf("\n%s%d%s\n", "Images out scope found: [", len(linksToImagesOutScope), "]")
	}

	if emailMatches != nil {
		fmt.Printf("\n%s%d%s\n", "Emails found: [", len(emailMatches), "]")
		fmt.Printf("%s\n", strings.Repeat("-", 12))
		for _, email := range emailMatches {
			fmt.Printf("%s\n", email)
		}
	} else {
		fmt.Printf("\n%s%d%s\n", "Emails found: [", len(emailMatches), "]")
	}

	fmt.Println()
}

func crawlHREFInScope(path string) {
	var targetURL url.URL
	targetURL.Scheme = startingURL.Scheme
	targetURL.Host = startingURL.Host
	targetURL.Path = path

	resp, err := webRequest(http.MethodGet, targetURL.String())
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		parsedHref, err := url.Parse(href)
		if err != nil {
			return
		}
		if strings.Contains(parsedHref.String(), "mailto:") {
			re := regexp.MustCompile(`([a-zA-Z0-9_\-\.]+)@([a-zA-Z0-9_\-\.]+)\.([a-zA-Z]{1,5})`)
			mailAddress := re.FindString(parsedHref.String())
			if mailNotExist(mailAddress) {
				emailMatches = append(emailMatches, mailAddress)
			}
		} else {
			if urlIsInScope(parsedHref) {
				if linkContains(fileExtensions, parsedHref.String()) {
					if isNotRepeated(linksToFilesInScope, parsedHref.String()) {
						linksToFilesInScope = append(linksToFilesInScope, parsedHref.String())
					}
				} else if linkContains(imageExtensions, parsedHref.String()) {
					if isNotRepeated(linksToImagesInScope, parsedHref.String()) {
						linksToImagesInScope = append(linksToImagesInScope, parsedHref.String())
					}
				} else {
					var URL paths
					URL.url = parsedHref
					fmt.Println("\nNew path to crawl: " + parsedHref.String())
					foundPaths = append(foundPaths, URL)
					crawlHREFOutOfScope(parsedHref.Path)
					crawlHREFInScope(parsedHref.Path)
				}
			} else {
				if urlIsOutOfScope(parsedHref) {
					if linkContains(fileExtensions, parsedHref.String()) {
						if isNotRepeated(linksToFilesOutScope, parsedHref.String()) {
							linksToFilesOutScope = append(linksToFilesOutScope, parsedHref.String())
						}
					} else if linkContains(imageExtensions, parsedHref.String()) {
						if isNotRepeated(linksToImagesOutScope, parsedHref.String()) {
							linksToImagesOutScope = append(linksToImagesOutScope, parsedHref.String())
						}
					}
				}
			}
		}
	})
}

func crawlHREFOutOfScope(path string) {
	var targetURL url.URL
	targetURL.Scheme = startingURL.Scheme
	targetURL.Host = startingURL.Host
	targetURL.Path = path
	foundExtURL = nil

	resp, err := webRequest(http.MethodGet, targetURL.String())
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		parsedURL, err := url.Parse(href)
		if err != nil {
			return
		}
		if uniqExtURL(parsedURL) {
			uniqueExtURL = append(uniqueExtURL, parsedURL.String())
		}
		if urlIsOutOfScope(parsedURL) {
			fmt.Printf("%20s %s\n", ">", parsedURL.String())
			foundExtURL = append(foundExtURL, parsedURL.String())
		}
	})
}

func webRequest(method, URL string) (*http.Response, error) {
	var (
		trCfg = &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: timeout,
			}).Dial,
			TLSHandshakeTimeout: timeout,
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS11,
				InsecureSkipVerify: true,
			},
		}
		jar, _     = cookiejar.New(nil)
		httpClient = &http.Client{
			Timeout:   timeout,
			Transport: trCfg,
			Jar:       jar,
		}
	)

	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header = map[string][]string{
		"User-Agent": {"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"},
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func urlIsInScope(tempURL *url.URL) bool {
	if tempURL.Path == "" {
		return false
	}
	if tempURL.Host != "" && tempURL.Host != startingURL.Host {
		return false
	}
	for _, existingPath := range foundPaths {
		if existingPath.url.Path == tempURL.Path {
			return false
		}
	}
	return true
}

func urlIsOutOfScope(tempURL *url.URL) bool {
	if tempURL.Path == "" {
		return false
	}
	for _, existingURL := range foundExtURL {
		if existingURL == tempURL.String() {
			return false
		}
	}
	if tempURL.Host != "" && tempURL.Host != startingURL.Host {
		return true
	}
	return false
}

func uniqExtURL(tempURL *url.URL) bool {
	if tempURL.Path == "" {
		return false
	}
	for _, uniqExtURL := range uniqueExtURL {
		if uniqExtURL == tempURL.String() {
			return false
		}
	}
	if tempURL.Host != "" && tempURL.Host != startingURL.Host {
		return true
	}
	return false
}

func mailNotExist(mailAddress string) bool {
	for _, match := range emailMatches {
		if match == mailAddress {
			return false
		}
	}
	return true
}

func linkContains(fileExtensionList []string, url string) bool {
	urlPieces := strings.Split(url, ".")
	if len(urlPieces) < 2 {
		return false
	}
	for _, ext := range fileExtensionList {
		if urlPieces[len(urlPieces)-1] == ext {
			return true
		}
	}
	return false
}

func isNotRepeated(links []string, href string) bool {
	unique := true
	for _, l := range links {
		if l == href {
			unique = false
			break
		} else {
			continue
		}
	}
	return unique
}
