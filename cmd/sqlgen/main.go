package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

type TemplateParam struct {
	PackageName string
	StructName  string
	Config      TableConfig
}

func main() {
	f := GetFlags()
	if err := f.Validate(); err != nil {
		panic(err)
	}

	configFile, err := os.ReadFile(f.ConfigPath)
	if err != nil {
		panic(err)
	}

	var yamlData YamlData
	if err := yaml.Unmarshal(configFile, &yamlData); err != nil {
		panic(err)
	}

	t, err := template.New("template").
		Funcs(template.FuncMap{
			"Join": func(s []string) string {
				return strings.Join(s, ", ")
			},
			"JoinAnd": func(s []string) string {
				return strings.Join(s, " AND ")
			}}).
		Parse(tmpl)
	if err != nil {
		panic(err)
	}

	if err := os.MkdirAll(f.OutputDir, os.ModePerm); err != nil {
		panic(err)
	}

	for _, table := range yamlData.Tables {
		if err := table.Validate(); err != nil {
			panic(err)
		}

		out := path.Join(f.OutputDir, getFileName(table.Name))
		outFile, err := os.Create(out)
		if err != nil {
			panic(err)
		}

		params := TemplateParam{
			PackageName: "tables",
			StructName:  getStructName(table.Name),
			Config:      table,
		}

		if err := t.Execute(outFile, params); err != nil {
			panic(err)
		}
	}
}

const structSuffix = "Table"

func getStructName(tableName string) string {
	sb := strings.Builder{}
	words := strings.Split(tableName, "_")
	for _, w := range words {
		sb.WriteString(cases.Title(language.English).String(w))
	}
	sb.WriteString(structSuffix)

	return sb.String()
}

func getFileName(tableName string) string {
	return tableName + "_gen.go"
}

type Flags struct {
	ConfigPath string
	OutputDir  string
}

func (f Flags) Validate() error {
	if len(f.ConfigPath) == 0 {
		return fmt.Errorf("missing config file")
	}

	if _, err := os.Stat(f.ConfigPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s: file does not exist", f.ConfigPath)
	}

	if len(f.OutputDir) == 0 {
		return fmt.Errorf("missing output directory")
	}

	return nil
}

func GetFlags() Flags {
	configPath := flag.String("config", "", "path to config file (yaml)")
	outputDir := flag.String("out", "", "path to a folder where will be generated into")
	flag.Parse()

	return Flags{
		ConfigPath: *configPath,
		OutputDir:  *outputDir,
	}
}
