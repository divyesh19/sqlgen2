package code

import (
	"fmt"
	. "strings"
	"text/template"

	. "github.com/acsellers/inflections"
	"github.com/rickb777/sqlgen2/schema"
)

type View struct {
	Prefix   string
	Type     string
	Types    string
	Suffix   string
	Body1    []string
	Body2    []string
	Body3    []string
	Dialects []string
	Table    *schema.Table
}

func NewView(name, prefix string) View {
	return View{
		Prefix: prefix,
		Type:   name,
		Types:  Pluralize(name),
	}
}

func (v View) DbName() string {
	return ToLower(v.Types)
}

var funcMap = template.FuncMap{
	"q": func(s interface{}) string {
		return fmt.Sprintf("%q", s)
	},
	"ticked": func(s interface{}) string {
		return fmt.Sprintf("`\n%s\n`", s)
	},
}
