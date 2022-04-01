package main

const tmpl = `// This file was automatically generated. Please do not edit manually.

package {{ .PackageName }}

import (
	"fmt"
)

type {{ .StructName }} struct {
	tableName string
}

func New{{ .StructName }}(tableName string) {{ .StructName }} {
	return {{ .StructName }}{
		tableName: tableName,
	}
}

func (r {{ .StructName }}) CreateTable() string {
	return fmt.Sprintf(` + "`" + `
		CREATE TABLE IF NOT EXISTS %s
		({{ Join .Config.ColumnsDefinition }})
	` + "`" + `, r.tableName)
}

func (r {{ .StructName }}) Insert() string {
	return fmt.Sprintf(` + "`" + `
		INSERT INTO %s ({{ Join .Config.InsertColumns }})
		VALUES ({{ Join .Config.InsertColumnsParams }})
	` + "`" + `, r.tableName)
}

func (r {{ .StructName }}) Upsert() string {
	return fmt.Sprintf(` + "`" + `
		INSERT INTO %s ({{ Join .Config.InsertColumns }})
		VALUES ({{ Join .Config.InsertColumnsParams }})
		ON CONFLICT ({{ Join .Config.UniqueColumns }})
		DO UPDATE
		SET {{ Join .Config.UpsertSet }}
	` + "`" + `, r.tableName)
}

func (r {{ .StructName }}) Delete() string {
	return fmt.Sprintf(` + "`" + `
		DELETE FROM %s
		WHERE {{ JoinAnd .Config.WhereConditions }}
	` + "`" + `, r.tableName)
}
`