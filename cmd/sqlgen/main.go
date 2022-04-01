package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

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
	err = yaml.Unmarshal(configFile, &yamlData)
	if err != nil {
		panic(err)
	}

	t, err := template.New("template").
		Funcs(template.FuncMap{
			"Join": func(s []string) string {
				return strings.Join(s, ",")
			},
			"JoinAnd": func(s []string) string {
				return strings.Join(s, " AND ")
			}}).
		Parse(tmpl)
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(f.OutputDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	for _, table := range yamlData.Tables {
		err := table.Validate()
		if err != nil {
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

		err = t.Execute(outFile, params)
		if err != nil {
			panic(err)
		}
	}
}

const structSuffix = "Table"

func getStructName(tableName string) string {
	sb := strings.Builder{}
	words := strings.Split(tableName, "_")
	for _, w := range words {
		sb.WriteString(strings.ToUpper(w[0:1]))
		sb.WriteString(w[1:])
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
