package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOutputFileName(t *testing.T) {
	a := assert.New(t)
	a.Equal("input.gen.go", outputFileName("input.go"))
	a.Equal("input.txt.gen.go", outputFileName("input.txt"))
}
