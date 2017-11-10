package httpsig

import (
	"bytes"
	"net/textproto"
	"strings"
)

type writer struct {
	bytes.Buffer
	headers *[]string
}

func (w *writer) write(name string, values ...string) {
	var validValues []string
	for _, value := range values {
		value = headerNewlineToSpace.Replace(value)
		value = textproto.TrimString(value)
		if value != "" {
			validValues = append(validValues, value)
		}
	}

	if len(validValues) == 0 {
		return
	}

	name = strings.ToLower(name)

	if w.headers == nil {
		w.headers = &[]string{}
	}

	*w.headers = append(*w.headers, name)

	// the signature lines are, well, lines
	if w.Len() > 0 {
		w.WriteByte('\n')
	}

	w.WriteString(name + ": ")

	for i, value := range validValues {
		w.WriteString(value)

		// each value for the is comma+space separated
		if i < len(validValues)-1 {
			w.WriteString(", ")
		}
	}
}
