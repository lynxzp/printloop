package webserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "plain name", input: "model.gcode", expected: "model.gcode"},
		{name: "unix path traversal", input: "../../etc/passwd", expected: "passwd"},
		{name: "windows full path", input: `C:\Users\bob\part.gcode`, expected: "part.gcode"},
		{name: "mixed separators", input: `..\..//model.gcode`, expected: "model.gcode"},
		{name: "empty name", input: "", expected: "input.gcode"},
		{name: "dot", input: ".", expected: "input.gcode"},
		{name: "double dot", input: "..", expected: "input.gcode"},
		{name: "trailing separator", input: "dir/", expected: "input.gcode"},
		{name: "spaces and symbols kept", input: "test file & symbols.gcode", expected: "test file & symbols.gcode"},
		{name: "cyrillic kept", input: "модель.gcode", expected: "модель.gcode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, sanitizeFileName(tt.input))
		})
	}
}

func TestResultFileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		iterations int64
		expected   string
	}{
		{name: "regular gcode", input: "model.gcode", iterations: 5, expected: "model_x5.gcode"},
		{name: "no extension", input: "model", iterations: 3, expected: "model_x3"},
		{name: "multiple dots", input: "my.model.gcode", iterations: 2, expected: "my.model_x2.gcode"},
		{name: "max iterations", input: "part.gcode", iterations: 10000, expected: "part_x10000.gcode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, resultFileName(tt.input, tt.iterations))
		})
	}
}
