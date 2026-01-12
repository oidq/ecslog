package ecslog

import (
	"reflect"
	"slices"
	"testing"
)

var testSortAttrsData = []struct {
	name   string
	input  []string
	output []string
}{
	{
		name: "Basic",
		input: []string{
			"log.data",
			"source.func",
			"log.input",
		},
		output: []string{
			"source.func",
			"log.input",
			"log.data",
		},
	},
	{
		name: "Suffix",
		input: []string{
			"log.data",
			"source.func",
			"log",
		},
		output: []string{
			"source.func",
			"log.data",
			"log",
		},
	},
}

func TestSortAttrs(t *testing.T) {
	for _, data := range testSortAttrsData {
		t.Run(data.name, func(t *testing.T) {
			var inputCopy []string
			inputCopy = append(inputCopy, data.input...)

			slices.SortStableFunc(inputCopy, isEarlierKey)

			if !reflect.DeepEqual(data.output, inputCopy) {
				t.Errorf("invalid comparison: expected %#v, got %#v", data.output, inputCopy)
			}
		})
	}
}
