package code

import (
	. "fmt"
	"io"

	"github.com/rickb777/sqlgen2/schema"
	"strings"
)

func WriteRowsFunc(w io.Writer, view View) {

	for i, field := range view.Table.Fields {
		if field.Tags.Skip {
			continue
		}

		nullable := field.Type.NullableValue()

		if field.Type.IsScanner || field.Encode == schema.ENCDRIVER {
			nullable = ""
		}

		// temporary variable declaration
		l1 := ""
		switch field.Encode {
		case schema.ENCJSON, schema.ENCTEXT:
			l1 = Sprintf("\t\tvar v%d %s\n", i, "[]byte")
		case schema.ENCDRIVER:
			l1 = Sprintf("\t\tvar v%d %s\n", i, field.Type.Type())
		default:
			l1 = Sprintf("\t\tvar v%d %s\n", i, field.Type.Type())
			if nullable != "" {
				l1 = Sprintf("\t\tvar v%d sql.Null%s\n", i, nullable)
			}
		}
		view.Body1 = append(view.Body1, l1)

		// variable scanning
		l2 := Sprintf("&v%d", i)
		view.Body2 = append(view.Body2, l2)

		switch field.Encode {
		case schema.ENCJSON:
			l3 := Sprintf("\t\terr = json.Unmarshal(v%d, &v.%s)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n",
				i, field.JoinParts(0, "."))
			view.Body3 = append(view.Body3, l3)
		case schema.ENCTEXT:
			l3 := Sprintf("\t\terr = encoding.UnmarshalText(v%d, &v.%s)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n",
				i, field.JoinParts(0, "."))
			view.Body3 = append(view.Body3, l3)
		default:
			if nullable != "" {
				l3a := Sprintf("\t\tif v%d.Valid {\n", i)
				l3b := Sprintf("\t\t\ta := %s(v%d.%s)\n", field.Type.Type(), i, nullable)
				if field.Type.Name == strings.ToLower(nullable){
					l3b = Sprintf("\t\t\ta := v%d.%s\n", i, nullable)
				}
				l3c := Sprintf("\t\t\tv.%s = &a\n", field.JoinParts(0, "."))
				l3d := "\t\t}\n"
				view.Body3 = append(view.Body3, l3a)
				view.Body3 = append(view.Body3, l3b)
				view.Body3 = append(view.Body3, l3c)
				view.Body3 = append(view.Body3, l3d)
			} else {
				amp := ""
				if field.Type.IsPtr {
					amp = "&"
				}
				l3 := Sprintf("\t\tv.%s = %sv%d\n", field.JoinParts(0, "."), amp, i)
				view.Body3 = append(view.Body3, l3)
			}
		}
	}

	must(tScanRows.Execute(w, view))
}
