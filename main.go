// Crawl a website, breadth-first, listing every local and external paths found in same host
package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type paths struct {
	url *url.URL
}

var (
	foundExtURL, uniqueExtURL []string
	foundPaths                []paths
	startingURL               *url.URL
	timeout                   time.Duration
)

func main() {
	// Load command line arguments
	if len(os.Args) != 3 {
		fmt.Println("\n[Simple Breadth-First Web Spider]")
		fmt.Println("\nUsage:   " + os.Args[0] + " <startingURL>" + " <timeout(secs)>")
		fmt.Println("\nExample: " + os.Args[0] + " https://www.example.com" + " 10\n")
		os.Exit(1)
	}
	foundExtURL = make([]string, 0)
	t, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal("Error parsing timeout. ", err)
	}
	timeout = time.Duration(time.Duration(t) * time.Second)
	// Parse starting URL
	startingURL, err = url.Parse(os.Args[1])
	if err != nil {
		log.Fatal("Error parsing starting URL. ", err)
	}
	fmt.Printf("\n%s%s%s\n", "Starting Simple Breadth-First Web Crawling for: [", startingURL.String(), "]")

	crawlURLInScope(startingURL.Path)
	// Print a summary
	fmt.Printf("\n%s%d%s\n", "Local paths: [", len(foundPaths), " found]")
	fmt.Printf("%s\n", strings.Repeat("-", 12))
	for _, path := range foundPaths {
		fmt.Printf("%s\n", path.url.String())
	}
	fmt.Printf("\n%s%d%s\n", "External paths: [", len(uniqueExtURL), " found]")
	fmt.Printf("%s\n", strings.Repeat("-", 15))
	for _, uniqExtURL := range uniqueExtURL {
		fmt.Printf("%s\n", uniqExtURL)
	}
	fmt.Println()
}

func crawlURLInScope(path string) {
	// Create a temporary URL object for this request
	var targetURL url.URL
	targetURL.Scheme = startingURL.Scheme
	targetURL.Host = startingURL.Host
	targetURL.Path = path
	// Fetch the URL with a timeout, ignore certificate verification and parse to goquery doc
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
	// Find all links and crawl if new path on same host
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		parsedURL, err := url.Parse(href)
		if err != nil {
			return
		}
		if urlIsInScope(parsedURL) {
			var URL paths
			URL.url = parsedURL
			fmt.Println("\nNew path to crawl: " + parsedURL.String())
			foundPaths = append(foundPaths, URL)
			crawlURLOutOfScope(parsedURL.Path)
			crawlURLInScope(parsedURL.Path)
		}
	})
}

func crawlURLOutOfScope(path string) {
	// clearing slice...
	foundExtURL = nil
	// Create a temporary URL object for this request
	var targetURL url.URL
	targetURL.Scheme = startingURL.Scheme
	targetURL.Host = startingURL.Host
	targetURL.Path = path
	// Fetch the URL with a timeout, ignore certificate verification and parse to goquery doc
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
	// Find all links and crawl if new path is found on same host
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
			return false // Match
		}
	}
	if tempURL.Host != "" && tempURL.Host != startingURL.Host {
		return true
	}
	return false
}
