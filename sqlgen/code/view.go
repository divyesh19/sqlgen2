package code

import (
	"fmt"
	. "strings"
	"text/template"

	. "github.com/acsellers/inflections"
	"github.com/rickb777/sqlgen2/schema"
	"bitbucket.org/pkg/inflect"
	"github.com/rickb777/sqlgen2/constraint"
)

type View struct {
	Prefix     string
	Type       string
	Types      string
	Thing      string
	Interface1 string
	Interface2 string
	List       string
	Suffix     string
	Body1      []string
	Body2      []string
	Body3      []string
	Dialects   []string
	Table      *schema.TableDescription
	Setter     *schema.Field
}

func NewView(name, prefix, list string) View {
	if list == "" {
		list = fmt.Sprintf("[]*%s", name)
	}
	return View{
		Prefix:     prefix,
		Type:       name,
		Types:      Pluralize(name),
		Thing:      "Table",
		Interface1: "sqlgen2.Table",
		Interface2: "sqlgen2.Table",
		List:       list,
		Dialects:   schema.DialectNames(),
	}
}

func (v View) DbName() string {
	return ToLower(v.Types)
}

func (v View) CamelName() string {
	return v.Prefix + inflect.Camelize(v.Table.Type)
}

func (v View) Constraints() (list constraint.Constraints) {
	for _, f := range v.Table.Fields {
		if f.Tags.ForeignKey != "" {
			slice := Split(f.Tags.ForeignKey, ".")
			c := constraint.FkConstraintOn(f.SqlName).
				RefersTo(slice[0], slice[1]).
				OnUpdate(constraint.Consequence(f.Tags.OnUpdate)).
				OnDelete(constraint.Consequence(f.Tags.OnDelete))
			list = append(list, c)
		}
	}
	return list
}

var funcMap = template.FuncMap{
	"q": func(s interface{}) string {
		return fmt.Sprintf("%q", s)
	},
	"camel": func(s interface{}) string {
		return inflect.Camelize(fmt.Sprintf("%s", s))
	},
	"ticked": func(s interface{}) string {
		return fmt.Sprintf("`\n%s\n`", s)
	},
	"title": func(s interface{}) string {
		return Title(fmt.Sprintf("%s", s))
	},
}
