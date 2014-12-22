package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerator(t *testing.T) {
	type testcase struct {
		// input
		file    []byte
		structs []string

		//expect
		error           bool
		import_packages []string
		package_name    string
		methods         []methodProperty
		struct_length   int
	}

	a := assert.New(t)
	cases := []testcase{
		{
			[]byte(`package main
import "fmt"

type TestEntity struct {
	TestCol string
}`),
			[]string{"TestEntity"},
			false,
			[]string{},
			"main",
			[]methodProperty{},
			1,
		}, {
			[]byte(`package main
import (
	"time"
	"github.com/coopernurse/gorp"
	"fmt"
	cvss "github.com/umisama/go-cvss"
)

type TestEntity struct {
	TestCol		time.Time
	TimeCol		gorp.NullTime
	VectorCol	cvss.Vector
	isNew		bool
}`),
			[]string{"TestEntity"},
			false,
			[]string{"time", "github.com/coopernurse/gorp", "github.com/umisama/go-cvss"},
			"main",
			[]methodProperty{},
			3,
		}, {
			[]byte(`import "fmt"

type TestEntity struct {
	TestCol string
}`),
			[]string{"TestEntity"},
			true,
			nil,
			"",
			[]methodProperty{},
			1,
		}, {
			[]byte(`package main
import (
	"time"
	"github.com/coopernurse/gorp"
	"fmt"
	cvss "github.com/umisama/go-cvss"
)

type TestEntity struct {
	TestCol		time.Time
	TimeCol		gorp.NullTime
	VectorCol	cvss.Vector
	isNew		bool
}

func (t *TestEntity) checkTest(b string) error {
	return true
}

func (t *TestEntity) OtherFunc(b fmt.TheType) fmt.TheType {
	return true
}

func (t *OtherEntity) checkOtherTest(b string) bool {
	return true
}`),
			[]string{"TestEntity"},
			false,
			[]string{"time", "github.com/coopernurse/gorp", "github.com/umisama/go-cvss", "fmt"},
			"main",
			[]methodProperty{
				{"checkTest", []string{"error"}, []string{"string"}},
				{"OtherFunc", []string{"fmt.TheType"}, []string{"fmt.TheType"}},
			},
			3,
		},
	}

	for _, c := range cases {
		gen, err := newGenerator(c.file, c.structs)
		a.Nil(err)
		a.NotNil(gen)

		err = gen.Run()
		a.Equal(c.error, err != nil)
		if !c.error {
			a.Equal(c.import_packages, gen.imports)
			a.Equal(c.package_name, gen.package_name)
			for key, method := range c.methods {
				a.Equal(method, gen.props[0].Methods[key])
			}
			a.Equal(c.struct_length, len(gen.props[0].Fields))
		}
	}
}

func TestCreateSetter(t *testing.T) {
	a := assert.New(t)
	type testcase struct {
		// input
		obj *structProperty

		// expect
		funcstr string
	}

	var cases = []testcase{{
		obj: &structProperty{
			Name: "TestStruct",
			Fields: []fieldProperty{
				{"FieldCol", "Type"},
			},
		},
		funcstr: `// AUTO GENERATED
func (m *TestStruct) SetField (val Type) error {
	m.FieldCol = val
	return nil
}`,
	}, {
		obj: &structProperty{
			Name: "TestStruct",
			Fields: []fieldProperty{
				{"FieldCol", "Type"},
			},
			Methods: []methodProperty{
				{"checkField", []string{"Type"}, []string{"error"}},
			},
		},
		funcstr: `// AUTO GENERATED
func (m *TestStruct) SetField (val Type) error {
	if err := m.checkField(val); err != nil {
		return err
	}
	m.FieldCol = val
	return nil
}`,
	}}

	for _, c := range cases {
		a.Equal(c.funcstr, c.obj.createSetter(0))
	}
}
