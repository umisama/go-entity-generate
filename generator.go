package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
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
	Name    string
	Fields  []fieldProperty
	Methods []methodProperty
}

type fieldProperty struct {
	Name string
	Type string
}

type methodProperty struct {
	Name    string
	Results []string
	Params  []string
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

	m.package_name = astf.Name.Name
	return nil
}

func (m *Generator) Output() (io.Reader, error) {
	ret := m.createHeader()
	for _, prop := range m.props {
		ret += "\n\n" + prop.createInterface()
		for i := 0; i < prop.len(); i++ {
			ret += "\n\n" + prop.createGetter(i)
			ret += "\n\n" + prop.createSetter(i)
		}
	}
	return bytes.NewBufferString(ret), nil
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
		Name:    name,
		Fields:  make([]fieldProperty, 0),
		Methods: make([]methodProperty, 0),
	}

	for _, field := range src.Fields.List {
		switch t := field.Type.(type) {
		case fmt.Stringer:
			p.appendField(fieldProperty{
				Name: field.Names[0].Name,
				Type: t.String(),
			})
		case *ast.ArrayType:
			switch elem_t := t.Elt.(type) {
			case *ast.SelectorExpr:
				p.appendField(fieldProperty{
					Name: field.Names[0].Name,
					Type: elem_t.Sel.String(),
				})
			case *ast.Ident:
				p.appendField(fieldProperty{
					Name: field.Names[0].Name,
					Type: elem_t.String(),
				})
			}
		case *ast.SelectorExpr:
			sel := t.X.(*ast.Ident)
			p.appendField(fieldProperty{
				Name: field.Names[0].Name,
				Type: sel.Name + "." + t.Sel.Name,
			})
			m.addImportPath(getPackagePath(file, sel.Name))
		}
	}

	for _, decl := range file.Decls {
		fndecl, ok := decl.(*ast.FuncDecl)
		if !ok || fndecl.Recv == nil || len(fndecl.Recv.List) == 0 {
			continue // not method
		}
		if p.Name != fndecl.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name {
			continue // not for this type
		}

		name := fndecl.Name.String()
		params, rets := []string{}, []string{}
		for _, param := range fndecl.Type.Params.List {
			switch expr := param.Type.(type) {
			case *ast.Ident:
				params = append(params, expr.Name)
			case *ast.SelectorExpr:
				params = append(params, expr.X.(*ast.Ident).Name+"."+expr.Sel.Name)
				m.addImportPath(getPackagePath(file, expr.X.(*ast.Ident).Name))
			}
		}
		for _, ret := range fndecl.Type.Results.List {
			switch expr := ret.Type.(type) {
			case *ast.Ident:
				rets = append(rets, expr.Name)
			case *ast.SelectorExpr:
				rets = append(rets, expr.X.(*ast.Ident).Name+"."+expr.Sel.Name)
				m.addImportPath(getPackagePath(file, expr.X.(*ast.Ident).Name))
			}
		}

		p.appendMethod(methodProperty{
			Name:    name,
			Results: rets,
			Params:  params,
		})
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

	for _, v := range p.Methods {
		if v.Name == "check"+tmpl_dat["field_name"] {
			tmpl_dat["checkfunc"] = v.Name
		}
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

	methods := make([]methodProperty, 0)
	for _, m := range p.Methods {
		if !strings.HasPrefix(m.Name, "check") {
			methods = append(methods, m)
		}
	}

	tmpl_dat := map[string]interface{}{
		"interface_name": strings.TrimSuffix(strings.Title(p.Name), "Entity"),
		"fields":         fields,
		"methods":        methods,
	}

	buf := bytes.NewBuffer([]byte{})
	template.Must(template.New("tmpl").Parse(TMPL_INTERFACE)).Execute(buf, tmpl_dat)
	return buf.String()
}

func (p *structProperty) appendField(d fieldProperty) {
	if strings.ToUpper(string(d.Name[0])) != string(d.Name[0]) {
		return
	}

	p.Fields = append(p.Fields, d)
}

func (p *structProperty) appendMethod(d methodProperty) {
	p.Methods = append(p.Methods, d)
}

func getStructTypes(src *ast.File, struct_names []string) map[string]*ast.StructType {
	structs := make(map[string]*ast.StructType)

declloop:
	for _, decl := range src.Decls {
		gendecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range gendecl.Specs {
			typ, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue declloop
			}
			strtyp, ok := typ.Type.(*ast.StructType)
			if !ok {
				continue declloop
			}
			name := typ.Name.Name
			if isAnyOne(name, struct_names) {
				structs[name] = strtyp
			}
		}
	}

	return structs
}

func getPackagePath(src *ast.File, sel string) string {
	for _, imp := range src.Imports {
		path := strings.Trim(imp.Path.Value, "\"")

		name := ""
		if imp.Name != nil {
			name = imp.Name.String()
		} else {
			s := strings.Split(path, "/")
			name = s[len(s)-1]
		}

		if name == sel {
			return path
		}
	}
	return ""
}

func isAnyOne(src string, trg []string) bool {
	for _, s := range trg {
		if src == s {
			return true
		}
	}
	return false
}
