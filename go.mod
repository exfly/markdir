module github.com/exfly/markdir

go 1.22

toolchain go1.24.0

require github.com/russross/blackfriday v1.6.0

require (
	github.com/yuin/goldmark v1.7.10 // indirect
	go.abhg.dev/goldmark/wikilink v0.6.0 // indirect
)

replace (
	go.abhg.dev/goldmark/wikilink => ./third_part/goldmark-wikilink
)
