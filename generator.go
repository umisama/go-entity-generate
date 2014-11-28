package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"strings"
	"text/template"
)

type Generator struct {
	input   []byte
	structs []string
	output  []byte

	props        []*structProperty
	package_name string
	imports      []string
}

type structProperty struct {
	Name   string
	Fields []fieldProperty
}

type fieldProperty struct {
	Name string
	Type string
}

func NewGenerator(input_file string, struct_names []string) (*Generator, error) {
	file, err := ioutil.ReadFile(input_file)
	if err != nil {
		return nil, err
	}

	return newGenerator(file, struct_names)
}

func newGenerator(input []byte, struct_names []string) (*Generator, error) {
	return &Generator{
		input:   input,
		structs: struct_names,

		props:        make([]*structProperty, 0),
		package_name: "",
		imports:      []string{},
	}, nil
}

func (m *Generator) Run() error {
	astf, err := parser.ParseFile(token.NewFileSet(), "generate.go", m.input, 0)
	if err != nil {
		return err
	}

	for name, typ := range getStructTypes(astf, m.structs) {
		m.props = append(m.props, m.NewStructProperty(name, astf, typ))
	}

	m.package_name = getPackageName(astf)
	return nil
}

func (m *Generator) Output() ([]byte, error) {
	ret := m.createHeader()
	for _, prop := range m.props {
		ret += "\n" + prop.createInterface()
		for i := 0; i < prop.len(); i++ {
			ret += "\n" + prop.createGetter(i)
			ret += "\n" + prop.createSetter(i)
		}
	}
	return []byte(ret), nil
}

func (m *Generator) Input() ([]byte, error) {
	return nil, nil
}

func (m *Generator) createHeader() string {
	tmpl_dat := map[string]interface{}{
		"package_name": m.package_name,
		"imports":      m.imports,
	}

	buf := bytes.NewBuffer([]byte{})
	template.Must(template.New("tmpl").Parse(TMPL_HEADER)).Execute(buf, tmpl_dat)
	return buf.String()
}

func (m *Generator) NewStructProperty(name string, file *ast.File, src *ast.StructType) *structProperty {
	p := &structProperty{
		Name:   name,
		Fields: make([]fieldProperty, 0),
	}

	for _, field := range src.Fields.List {
		switch t := field.Type.(type) {
		case fmt.Stringer:
			p.Fields = append(p.Fields, fieldProperty{
				Name: field.Names[0].Name,
				Type: t.String(),
			})
		case *ast.ArrayType:
			switch elem_t := t.Elt.(type) {
			case *ast.SelectorExpr:
				p.Fields = append(p.Fields, fieldProperty{
					Name: field.Names[0].Name,
					Type: elem_t.Sel.String(),
				})
			case *ast.Ident:
				p.Fields = append(p.Fields, fieldProperty{
					Name: field.Names[0].Name,
					Type: elem_t.String(),
				})
			}
		case *ast.SelectorExpr:
			sel := t.X.(*ast.Ident)
			p.Fields = append(p.Fields, fieldProperty{
				Name: field.Names[0].Name,
				Type: sel.Name + "." + t.Sel.Name,
			})
			m.addImportPath(getPackagePath(file, sel.Name))
		}
	}

	return p
}

func (m *Generator) addImportPath(path string) {
	for _, v := range m.imports {
		if v == path {
			return
		}
	}
	m.imports = append(m.imports, path)
	return
}

func (p *structProperty) len() int {
	return len(p.Fields)
}

func (p *structProperty) createGetter(index int) string {
	f := p.Fields[index]
	tmpl_dat := map[string]string{
		"struct_name":    p.Name,
		"field_name":     strings.TrimSuffix(f.Name, "Col"),
		"type_name":      f.Type,
		"field_realname": f.Name,
	}

	buf := bytes.NewBuffer([]byte{})
	template.Must(template.New("tmpl").Parse(TMPL_GETTER)).Execute(buf, tmpl_dat)
	return buf.String()
}

func (p *structProperty) createSetter(index int) string {
	f := p.Fields[index]
	tmpl_dat := map[string]string{
		"struct_name":    p.Name,
		"field_name":     strings.TrimSuffix(f.Name, "Col"),
		"type_name":      f.Type,
		"field_realname": f.Name,
	}

	buf := bytes.NewBuffer([]byte{})
	template.Must(template.New("tmpl").Parse(TMPL_SETTER)).Execute(buf, tmpl_dat)
	return buf.String()
}

func (p *structProperty) createInterface() string {
	fields := make([]map[string]string, 0)
	for _, f := range p.Fields {
		if strings.HasSuffix(f.Name, "Col") {
			fields = append(fields, map[string]string{
				"field_name": strings.TrimSuffix(f.Name, "Col"),
				"type_name":  f.Type,
			})
		}
	}

	tmpl_dat := map[string]interface{}{
		"interface_name": strings.TrimSuffix(strings.Title(p.Name), "Entity"),
		"fields":         fields,
	}

	buf := bytes.NewBuffer([]byte{})
	template.Must(template.New("tmpl").Parse(TMPL_INTERFACE)).Execute(buf, tmpl_dat)
	return buf.String()
}

func getStructTypes(src *ast.File, struct_names []string) map[string]*ast.StructType {
	structs := make(map[string]*ast.StructType)
	fn := func(n ast.Node) bool {
		if typspec, ok := n.(*ast.TypeSpec); ok {
			if st, ok := typspec.Type.(*ast.StructType); ok {
				name := typspec.Name.Name
				if isAnyOne(name, struct_names) {
					structs[name] = st
				}
			}
		}
		return true
	}

	ast.Inspect(src, fn)
	return structs
}

func getPackagePath(src *ast.File, sel string) string {
	ret := ""
	fn := func(n ast.Node) bool {
		if typspec, ok := n.(*ast.ImportSpec); ok {
			path := strings.Trim(typspec.Path.Value, "\"")

			name := ""
			if typspec.Name != nil {
				name = typspec.Name.String()
			} else {
				s := strings.Split(path, "/")
				name = s[len(s)-1]
			}

			if name == sel {
				ret = path
				return false
			}
		}
		return true
	}

	ast.Inspect(src, fn)
	return ret
}

func getPackageName(src *ast.File) string {
	return src.Name.Name
}

func isAnyOne(src string, trg []string) bool {
	for _, s := range trg {
		if src == s {
			return true
		}
	}
	return false
}
