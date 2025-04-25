package main

import (
	"errors"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/wikilink"
)

var bind = flag.String("bind", "127.0.0.1:19000", "port to run the server on")

func main() {
	flag.Parse()

	httpdir := http.Dir(".")
	handler := renderer{httpdir, http.FileServer(httpdir)}

	log.Println("Serving on http://" + *bind)
	log.Fatal(http.ListenAndServe(*bind, handler))
}

var outputTemplate = template.Must(template.New("base").Parse(`
<html>
  <head>
	<meta charset="utf-8">
    <title>{{ .Path }}</title>
	<link rel="stylesheet" href="/index.css">
	<style>
    .markdown-body {
        min-width: 200px;
        max-width: 790px;
        margin: 0 auto;
        padding: 30px;
    }
  </style>
  </head>
  <body class="markdown-body">
    {{ .Body }}
  </body>
</html>
`))

type renderer struct {
	d http.Dir
	h http.Handler
}

func (r renderer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !strings.HasSuffix(req.URL.Path, ".md") && !strings.HasSuffix(req.URL.Path, "/guide") {
		r.h.ServeHTTP(rw, req)
		return
	}

	// net/http is already running a path.Clean on the req.URL.Path,
	// so this is not a directory traversal, at least by my testing
	var pathErr *os.PathError
	input, err := ioutil.ReadFile("." + req.URL.Path)
	if errors.As(err, &pathErr) {
		http.Error(rw, http.StatusText(http.StatusNotFound)+": "+req.URL.Path, http.StatusNotFound)
		log.Printf("file not found: %s", err)
		return
	}

	if err != nil {
		http.Error(rw, "Internal Server Error: "+err.Error(), 500)
		log.Printf("Couldn't read path %s: %v (%T)", req.URL.Path, err, err)
		return
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			&wikilink.Extender{
				Resolver: defaultResolver{},
			},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(input, &buf); err != nil {
		panic(err)
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")

	outputTemplate.Execute(rw, struct {
		Path string
		Body template.HTML
	}{
		Path: req.URL.Path,
		Body: template.HTML(buf.String()),
	})

}

var (
	_html = []byte(".md")
	_hash = []byte{'#'}
)

type defaultResolver struct{}

func (defaultResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	dest := make([]byte, len(n.Target)+len(_html)+len(_hash)+len(n.Fragment))

	var i int
	if len(n.Target) > 0 {
		i += copy(dest, n.Target)
		if filepath.Ext(string(n.Target)) == "" {
			i += copy(dest[i:], _html)
		}
	}
	if len(n.Fragment) > 0 {
		i += copy(dest[i:], _hash)
		i += copy(dest[i:], n.Fragment)
	}

	return dest[:i], nil
}
