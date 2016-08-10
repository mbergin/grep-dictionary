package grepdict

import (
	"bufio"
	"fmt"
	"github.com/gorilla/handlers"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"html/template"
	"net/http"
	"os"
	"regexp"
)

// wordCache is an in-memory cache of our word list
var wordCache []string

var templates = template.Must(template.ParseGlob("grep-dictionary.html"))

func init() {
	http.Handle("/", handlers.CompressHandler(http.HandlerFunc(grepHandler)))
}

type match struct {
	Before string
	Match  string
	After  string
}

// grepDictionary filters words by the regular expression pattern.
// It returns an error if the regular expression is not valid.
func grepDictionary(pattern string, words []string) ([]match, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	var matches []match
	for _, word := range words {
		if loc := re.FindStringIndex(word); loc != nil {
			m := match{Before: word[:loc[0]],
				Match: word[loc[0]:loc[1]],
				After: word[loc[1]:]}
			matches = append(matches, m)
		}
	}
	return matches, nil
}

// grepHandler handles the HTTP request for the query page.
// If the query string has the variable pattern set, it renders
// the word list search results.
func grepHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	// Load word list or log an error
	words, err := getWords()
	if err != nil {
		log.Errorf(ctx, "%+v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	// Get pattern from query string
	pattern := r.URL.Query().Get("pattern")

	data := struct {
		Pattern   string
		Matches   []match
		Error     string
		Highlight bool
	}{
		Pattern:   pattern,
		Matches:   nil,
		Error:     "",
		Highlight: r.URL.Query().Get("highlight") == "on",
	}

	// If pattern is present, search the word list
	if pattern != "" {
		matches, err := grepDictionary(pattern, words)
		if err == nil {
			data.Matches = matches
		} else {
			data.Error = err.Error()
		}
	}

	// Render the template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age: 86400")
	w.Header().Set("Etag", fmt.Sprintf("%v", appengine.VersionID(ctx)))
	if err := templates.ExecuteTemplate(w, "grep-dictionary.html", data); err != nil {
		log.Errorf(ctx, "%v", err)
	}
}

// getWords returns the cached word list, or loads it.
func getWords() ([]string, error) {
	if wordCache == nil {
		words, err := readLines("en_GB-large.txt")
		if err != nil {
			return nil, err
		}
		wordCache = words
	}
	return wordCache, nil
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
