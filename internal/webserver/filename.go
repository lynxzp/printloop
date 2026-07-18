package webserver

import (
	"fmt"
	"path"
	"strings"
)

const fallbackFileName = "input.gcode"

// sanitizeFileName reduces a client-supplied file name to a safe base name.
// Browsers may send a full path (Windows: `C:\dir\part.gcode`) and a crafted
// request may contain `../` segments, so everything up to the last path
// separator is dropped.
func sanitizeFileName(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}

	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return fallbackFileName
	}

	return name
}

// resultFileName inserts an _xN suffix before the extension:
// "model.gcode" with 5 iterations becomes "model_x5.gcode".
func resultFileName(name string, iterations int64) string {
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)

	return fmt.Sprintf("%s_x%d%s", base, iterations, ext)
}
