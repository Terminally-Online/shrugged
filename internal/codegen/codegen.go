package codegen

import (
	"fmt"

	"github.com/terminally-online/shrugged/internal/parser"
)

type Generator interface {
	Generate(schema *parser.Schema, outDir string) error
	Language() string
}

var generators = make(map[string]Generator)

func Register(g Generator) {
	generators[g.Language()] = g
}

func Get(language string) (Generator, error) {
	g, ok := generators[language]
	if !ok {
		return nil, fmt.Errorf("unknown language: %s", language)
	}
	return g, nil
}

func Languages() []string {
	var langs []string
	for lang := range generators {
		langs = append(langs, lang)
	}
	return langs
}
