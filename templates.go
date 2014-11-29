package main

const (
	TMPL_HEADER = `package {{.package_name}}

import ({{range .imports}}
	"{{.}}"{{end}}
)`
	TMPL_GETTER = `// AUTO GENERATED
func (m *{{.struct_name}}) {{.field_name}}(){{.type_name}} {
	return m.{{.field_realname}}
}`
	TMPL_SETTER = `// AUTO GENERATED
func (m *{{.struct_name}}) Set{{.field_name}} (val {{.type_name}}) error {
	{{if .checkfunc}}if err := m.{{.checkfunc}}(val); err != nil {
		return err
	}
	{{end}}m.{{.field_realname}} = val
	return nil
}`
	TMPL_INTERFACE = `// AUTO GENERATED
type {{.interface_name}} interface { {{range .fields}}
	{{.field_name}}(){{.type_name}}
	Set{{.field_name}}(val {{.type_name}}) error{{end}}
}`
)
