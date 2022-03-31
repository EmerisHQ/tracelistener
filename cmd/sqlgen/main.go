package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"
)

const tmpl = `
// This file was automatically generated. Please do not edit manually.
// {{ .Timestamp }}

package {{ .PackageName }}

type {{ .StructName }} struct {
	tableName string
}

func New{{ .StructName }}(tableName string) {
	return {{ .StructName }}{
		tableName: tableName,
	}
}

func (r {{ .StructName }}) CreateTable() string {
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ({{ Join .Config.ColumnsDefinition }})", r.tableName)
}

func (r {{ .StructName }}) Insert() string {
	return fmt.Sprintf("INSERT INTO %s ({{ Join .Config.InsertColumns }}) VALUES ({{ Join .Config.InsertColumnsParams }})", r.tableName)
}

func (r {{ .StructName }}) Upsert() string {
	return r.Insert() + " ON CONFLICT ({{ Join .Config.UniqueColumns }}) DO UPDATE SET {{ Join .Config.UpsertSet }}"
}
`

type YamlData struct {
	Tables []TableConfig
}

type TableConfig struct {
	Name    string
	Columns []struct {
		Name     string
		Type     string
		Primary  bool
		Default  bool
		Nullable bool
	}
	UniqueColumns []string `yaml:"unique_columns,flow"`
}

func (t TableConfig) ColumnsDefinition() []string {
	res := make([]string, 0, len(t.Columns))
	for _, c := range t.Columns {
		def := c.Name + " " + c.Type
		if c.Primary {
			def += " PRIMARY KEY"
		}
		if !c.Primary && !c.Nullable {
			def += " NOT NULL"
		}
		res = append(res, def)
	}

	if len(t.UniqueColumns) > 0 {
		res = append(res, "UNIQUE ("+strings.Join(t.UniqueColumns, ",")+")")
	}

	return res
}

func (t TableConfig) InsertColumns() []string {
	res := make([]string, 0, len(t.Columns))
	for _, c := range t.Columns {
		if c.Default {
			continue
		}
		res = append(res, c.Name)
	}
	return res
}

func (t TableConfig) InsertColumnsParams() []string {
	cols := t.InsertColumns()
	res := make([]string, 0, len(cols))
	for _, c := range cols {
		res = append(res, ":"+c)
	}
	return res
}

func (t TableConfig) UpsertSet() []string {
	insertCols := t.InsertColumns()
	res := make([]string, 0, len(insertCols))
	for _, c := range insertCols {
		res = append(res, fmt.Sprintf("%s = EXCLUDED.%s", c, c))
	}
	return res
}

type TemplateParam struct {
	PackageName string
	StructName  string
	Timestamp   time.Time
	Config      TableConfig
}

func main() {
	f, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	var yamlData YamlData
	err = yaml.Unmarshal(f, &yamlData)
	if err != nil {
		panic(err)
	}

	t, err := template.New("template").
		Funcs(template.FuncMap{
			"Join": func(s []string) string {
				return strings.Join(s, ",")
			}}).
		Parse(tmpl)
	if err != nil {
		panic(err)
	}

	params := TemplateParam{
		PackageName: "query",
		StructName:  "AuthRow",
		Timestamp:   time.Now(),
		Config:      yamlData.Tables[0],
	}

	err = t.Execute(os.Stdout, params)
	if err != nil {
		panic(err)
	}
}
