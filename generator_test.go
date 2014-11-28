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
		}, {
			[]byte(`import "fmt"

type TestEntity struct {
	TestCol string
}`),
			[]string{"TestEntity"},
			true,
			nil,
			"",
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
		}
	}
}
