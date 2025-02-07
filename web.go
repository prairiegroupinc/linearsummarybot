package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

//go:embed views/layout.html views/report.html
var viewsFS embed.FS

var (
	layoutTmpl = template.Must(template.ParseFS(viewsFS, "views/layout.html"))
	reportTmpl = template.Must(template.ParseFS(viewsFS, "views/report.html"))
)

type PageData struct {
	Title   string
	Content template.HTML
}

type ReportPageData struct {
	Report     *Report
	HasOrphans bool
}

func startWeb(listenAddr string) {
	http.HandleFunc("/", serveHTMLReport)
	http.HandleFunc("/report.txt", serveTextReport)
	log.Printf("Listening on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func serveTextReport(w http.ResponseWriter, r *http.Request) {
	report, err := buildReport()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, formatTextReport(report))
}

func serveHTMLReport(w http.ResponseWriter, r *http.Request) {
	// Get the report
	report, err := buildReport()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = serveSpecificHTMLReport(w, report)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func serveSpecificHTMLReport(w http.ResponseWriter, report *Report) error {
	// Check if there are any orphans
	hasOrphans := false
	for _, md := range report.Months {
		if other, ok := md.Initiatives["Other"]; ok && len(other.Issues) > 0 {
			hasOrphans = true
			break
		}
	}

	// Render the report template
	var content strings.Builder
	err := reportTmpl.Execute(&content, ReportPageData{
		Report:     report,
		HasOrphans: hasOrphans,
	})
	if err != nil {
		return err
	}

	// Render the layout template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = layoutTmpl.Execute(w, PageData{
		Title:   "Linear Report",
		Content: template.HTML(content.String()),
	})
	if err != nil {
		return err
	}

	return nil
}
