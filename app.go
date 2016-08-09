package grepdict

import (
	"bufio"
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

// grepDictionary filters words by the regular expression pattern.
// It returns an error if the regular expression is not valid.
func grepDictionary(pattern string, words []string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, word := range words {
		if re.MatchString(word) {
			matches = append(matches, word)
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

	data := struct {
		Matches []string
		Error   string
	}{
		Matches: nil,
		Error:   "",
	}

	// If pattern is present, search the word list
	pattern := r.URL.Query().Get("pattern")
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
	w.Header().Set("Cache-Control", "max-age: 86400")
	w.Header().Set("Etag", appengine.VersionID(ctx))
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
