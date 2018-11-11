// Crawl a website, breadth-first, listing all unique paths in & out of scope + emails + files found...
package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
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

var foundExtURL, uniqueExtURL, emailMatches, linksToFilesInScope, linksToFilesOutScope []string

var (
	documentExtensions = []string{"doc", "docx", "pdf", "txt", "asp", "aspx", "key",
		"odp", "ods", "pps", "ppt", "pptx", "json", "csv", "xlr", "xls", "xlsx", "dat",
		"db", "dbf", "log", "mdb", "sav", "sql", "xml", "zip", "gz", "tar", "jar", "7z",
		"arj", "deb", "pkg", "rar", "rpm", "z", "bin", "dmg", "iso", "toast", "vcd", "apk",
		"bat", "cgi", "pl", "exe", "py", "wsf", "bak", "cab", "cfg", "cpl", "cur", "dll",
		"dmp", "drv", "ini", "msi", "sys", "tmp", "odt", "rtf", "tex", "wks", "wps", "wpd",
		"c", "cpp", "cs", "h", "java", "sh", "swift", "vb", "rss", "js", "jsp", "php", "cfm",
		"cer", "crt", "crl", "pem", "pfx", "p12", "csr", "p7b", "p7r", "spc", "der"}
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

	if emailMatches != nil {
		fmt.Printf("\n%s%d%s\n", "Emails found: [", len(emailMatches), "]")
		fmt.Printf("%s\n", strings.Repeat("-", 12))
		for _, match := range emailMatches {
			fmt.Println(match)
		}
	} else {
		fmt.Printf("\n%s%d%s\n", "Emails found: [", len(emailMatches), "]")
	}

	if linksToFilesInScope != nil {
		fmt.Printf("\n%s%d%s\n", "Files in scope found: [", len(linksToFilesInScope), "]")
		fmt.Printf("%s\n", strings.Repeat("-", 20))
		for _, files := range linksToFilesInScope {
			fmt.Println(files)
		}
	} else {
		fmt.Printf("\n%s%d%s\n", "Files in scope found: [", len(linksToFilesInScope), "]")
	}

	if linksToFilesOutScope != nil {
		fmt.Printf("\n%s%d%s\n", "Files out scope found: [", len(linksToFilesOutScope), "]")
		fmt.Printf("%s\n", strings.Repeat("-", 21))
		for _, files := range linksToFilesOutScope {
			fmt.Println(files)
		}
	} else {
		fmt.Printf("\n%s%d%s\n", "Files out scope found: [", len(linksToFilesOutScope), "]")
	}
	fmt.Println()
}

func crawlHREFInScope(path string) {
	var targetURL url.URL
	targetURL.Scheme = startingURL.Scheme
	targetURL.Host = startingURL.Host
	targetURL.Path = path
	
	transCfg := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	httpClient := &http.Client{Timeout: timeout, Transport: transCfg}
	response, err := httpClient.Get(targetURL.String())
	if err != nil {
		return
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
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
			re := regexp.MustCompile("([a-zA-Z0-9_\\-\\.]+)@([a-zA-Z0-9_\\-\\.]+)\\.([a-zA-Z]{1,5})")
			mailAddress := re.FindString(parsedHref.String())
			if mailNotExist(mailAddress) {
				emailMatches = append(emailMatches, mailAddress)
			}
		}
		if urlIsInScope(parsedHref) {
			if linkContainsDocument(parsedHref.String()) {
				linksToFilesInScope = append(linksToFilesInScope, parsedHref.String())
			}
		} else {
			if urlIsOutOfScope(parsedHref) {
				if linkContainsDocument(parsedHref.String()) {
					linksToFilesOutScope = append(linksToFilesOutScope, parsedHref.String())
				}
			}
		}
		if urlIsInScope(parsedHref) {
			var URL paths
			URL.url = parsedHref
			fmt.Println("\nNew path to crawl: " + parsedHref.String())
			foundPaths = append(foundPaths, URL)
			crawlHREFOutOfScope(parsedHref.Path)
			crawlHREFInScope(parsedHref.Path)
		}
	})
}

func crawlHREFOutOfScope(path string) {
	foundExtURL = nil
	var targetURL url.URL
	targetURL.Scheme = startingURL.Scheme
	targetURL.Host = startingURL.Host
	targetURL.Path = path

	transCfg := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	httpClient := &http.Client{Timeout: timeout, Transport: transCfg}
	response, err := httpClient.Get(targetURL.String())
	if err != nil {
		return
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
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

func linkContainsDocument(url string) bool {
	urlPieces := strings.Split(url, ".")
	if len(urlPieces) < 2 {
		return false
	}
	for _, extension := range documentExtensions {
		if urlPieces[len(urlPieces)-1] == extension {
			for _, match := range linksToFilesOutScope {
				if match == url {
					return false
				}
			}
			return true
		}
	}
	return false
}
