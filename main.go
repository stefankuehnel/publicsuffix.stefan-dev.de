package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

var (
	//go:embed template/*
	embededTemplateFileSystem embed.FS

	//go:embed static/*
	embededStaticFileSystem embed.FS
)

func redirectHttpHandler(url string) func(httpResponseWriter http.ResponseWriter, httpRequest *http.Request) {
	return func(httpResponseWriter http.ResponseWriter, httpRequest *http.Request) {
		http.Redirect(httpResponseWriter, httpRequest, url, 302)
	}
}

func indexHttpHandler(httpResponseWriter http.ResponseWriter, httpRequest *http.Request) {
	if httpRequest.URL.Path != "/" {
		http.NotFound(httpResponseWriter, httpRequest)
		return
	}

	template := template.Must(template.ParseFS(embededTemplateFileSystem, "template/index.html"))

	type TemplateData struct {
		DateTime string
		Year     int
	}

	templateData := TemplateData{
		DateTime: time.Now().Format("2006-01-02 15:04:05"),
		Year:     time.Now().Year(),
	}

	template.Execute(httpResponseWriter, templateData)
}

func publicSuffixHttpHandler(httpResponseWriter http.ResponseWriter, httpRequest *http.Request) {
	if httpRequest.URL.Path != "/publicsuffix" {
		http.NotFound(httpResponseWriter, httpRequest)
		return
	}

	type PublicSuffixHttpResponse struct {
		Domain       string `json:"domain"`
		PublicSuffix string `json:"publicSuffix"`
		IsManagedBy  string `json:"isManagedBy"`
	}

	publicSuffixHttpResponse := func(domain string) PublicSuffixHttpResponse {
		publicSuffix, isIcannManaged := publicsuffix.PublicSuffix(domain)

		isManagedBy := ""

		// See: https://pkg.go.dev/golang.org/x/net/publicsuffix#example-PublicSuffix-Manager
		if isIcannManaged {
			isManagedBy = "ICANN"
		} else if strings.IndexByte(publicSuffix, '.') >= 0 {
			isManagedBy = "PRIVATE_ENTITY"
		} else {
			isManagedBy = "NONE"
		}

		return PublicSuffixHttpResponse{
			Domain:       domain,
			PublicSuffix: publicSuffix,
			IsManagedBy:  isManagedBy,
		}
	}

	domain := httpRequest.URL.Query().Get("domain")

	httpResponseWriter.Header().Add("Content-Type", "application/json; charset=utf-8")

	if domain == "" {
		httpResponseWriter.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(httpResponseWriter).Encode(struct {
			ErrorCode    int    `json:"errorCode"`
			ErrorType    string `json:"errorType"`
			ErrorMessage string `json:"errorMessage"`
		}{
			ErrorCode:    http.StatusBadRequest,
			ErrorType:    http.StatusText(http.StatusBadRequest),
			ErrorMessage: "Malformed URL query parameter `domain`",
		})

		return
	}

	json.NewEncoder(httpResponseWriter).Encode(publicSuffixHttpResponse(domain))
}

func faviconHttpHandler(httpResponseWriter http.ResponseWriter, httpRequest *http.Request) {
	favicon, _ := embededStaticFileSystem.ReadFile("static/favicon.ico")

	httpResponseWriter.Write(favicon)
}

// See: https://pkg.go.dev/os#example-LookupEnv
func getEnv(key string, fallback string) string {
	value, exists := os.LookupEnv(key)

	if exists {
		return value
	}

	return fallback
}

func main() {
	// Static
	http.Handle("/static/", http.FileServer(http.FS(embededStaticFileSystem)))
	http.HandleFunc("/favicon.ico", faviconHttpHandler)

	// Dynamic
	http.HandleFunc("/publicsuffix", publicSuffixHttpHandler)

	// Redirects
	http.HandleFunc("/github", redirectHttpHandler("https://github.com/stefankuehnel/publicsuffix.stefan-dev.de"))

	// Templates
	http.HandleFunc("/", indexHttpHandler)

	port := getEnv("PORT", "80")

	log.Printf("listening on http://localhost:%s", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
