package main

import (
	"fmt"
	"strings"
)

type YamlData struct {
	Tables []TableConfig
}

type TableConfig struct {
	Name    string
	Columns []struct {
		Name         string
		Type         string
		Primary      bool
		SkipOnInsert bool `yaml:"skip_on_insert"`
		Nullable     bool
	}
	UniqueColumns []string `yaml:"unique_columns,flow"`
}

func (t TableConfig) Validate() error {
	var names map[string]bool
	for _, c := range t.Columns {
		if err := validateName(c.Name); err != nil {
			return fmt.Errorf("validating column name %s: %w", c.Name, err)
		}
		if len(c.Type) == 0 {
			return fmt.Errorf("column type cannot be empty")
		}
		if _, found := names[c.Name]; found {
			return fmt.Errorf("duplicate column name %s", c.Name)
		}
		if c.Primary && c.Nullable {
			return fmt.Errorf("primary column %s cannot be nullable", c.Name)
		}
		names[c.Name] = true
	}

	for _, c := range t.UniqueColumns {
		if _, found := names[c]; !found {
			return fmt.Errorf("unique column %s not defined in the columns section", c)
		}
	}

	return nil
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
		res = append(res, "UNIQUE ("+strings.Join(t.UniqueColumns, ", ")+")")
	}

	return res
}

func (t TableConfig) InsertColumns() []string {
	res := make([]string, 0, len(t.Columns))
	for _, c := range t.Columns {
		if c.SkipOnInsert {
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

func (t TableConfig) WhereConditions() []string {
	res := make([]string, 0, len(t.UniqueColumns))
	for _, c := range t.UniqueColumns {
		res = append(res, fmt.Sprintf("%s=:%s", c, c))
	}
	return res
}
