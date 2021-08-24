package template

import (
	"bytes"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

// ParseFile validates and parses passed as argument template file
func ParseFile(filename string, obj interface{}) ([]byte, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return ParseString(string(data), obj)
}

// ParseString validates and parses passed as argument template
func ParseString(strtmpl string, obj interface{}) ([]byte, error) {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Funcs(template.FuncMap{"spaces": spaces}).Parse(strtmpl)
	if err != nil {
		return nil, errors.Wrap(err, "error when parsing template")
	}
	err = tmpl.Execute(&buf, obj)
	if err != nil {
		return nil, errors.Wrap(err, "error when executing template")
	}
	return buf.Bytes(), nil
}

func spaces(n int, v string) string {
	pad := strings.Repeat(" ", n)
	return pad + strings.Replace(v, "\n", "\n"+pad, -1)
}
