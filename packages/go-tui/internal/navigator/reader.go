package navigator

import (
	"os"
	"regexp"
	"strings"
)

var (
	citationPattern   = regexp.MustCompile(`\[\^([\w-]+)\]`)
	definitionPattern = regexp.MustCompile(`^\[\^([\w-]+)\]:[ \t]+<?([^\s>]+)>?`)
)

const maxReadBytes = 5 * 1024 * 1024

// document is a file laid out for reading.
type document struct {
	title    string
	path     string
	slug     string
	evidence bool
	lines    []docLine
	targets  map[string]string
	marks    []citationMark
	error    string
}

// docLine is one wrapped line and the line of the file it came from.
type docLine struct {
	text   string
	source int
}

// citationMark is one citation on a wrapped line.
type citationMark struct {
	line  int
	label string
}

func loadDocument(path, title, slug string, evidence bool, width int) document {
	doc := document{title: title, path: path, slug: slug, evidence: evidence, targets: map[string]string{}}
	info, err := os.Stat(path)
	if err != nil {
		doc.error = "This file can't be opened."
		return doc
	}
	if info.Size() > maxReadBytes {
		doc.error = "This file is too large to read here. Press o to open it."
		return doc
	}
	data, err := os.ReadFile(path)
	if err != nil {
		doc.error = "This file can't be read."
		return doc
	}
	doc.layout(string(data), width)
	return doc
}

func (d *document) layout(text string, width int) {
	d.lines, d.marks = nil, nil
	for number, raw := range strings.Split(cleanText(text), "\n") {
		if match := definitionPattern.FindStringSubmatch(raw); match != nil {
			d.targets[match[1]] = match[2]
		}
		for _, part := range wrap(raw, width) {
			for _, match := range citationPattern.FindAllStringSubmatch(part, -1) {
				if !strings.HasPrefix(strings.TrimSpace(part), "[^"+match[1]+"]:") {
					d.marks = append(d.marks, citationMark{line: len(d.lines), label: match[1]})
				}
			}
			d.lines = append(d.lines, docLine{text: part, source: number + 1})
		}
	}
}

// lineFor finds the wrapped line that shows a line of the file.
func (d document) lineFor(source int) int {
	for index, line := range d.lines {
		if line.source >= source {
			return index
		}
	}
	return 0
}

func wrap(text string, width int) []string {
	if width < 20 {
		width = 20
	}
	if len([]rune(text)) <= width {
		return []string{text}
	}
	var lines []string
	current := ""
	for _, word := range strings.Fields(text) {
		for len([]rune(word)) > width {
			runes := []rune(word)
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			lines = append(lines, string(runes[:width]))
			word = string(runes[width:])
		}
		switch {
		case current == "":
			current = word
		case len([]rune(current))+1+len([]rune(word)) <= width:
			current += " " + word
		default:
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
