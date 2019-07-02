package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kortschak/utter"
	"github.com/rickb777/sqlapi/schema"
	"github.com/rickb777/sqlapi/types"
	. "github.com/rickb777/sqlgen2/code"
	. "github.com/rickb777/sqlgen2/load"
	"github.com/rickb777/sqlgen2/output"
	"github.com/rickb777/sqlgen2/parse"
	"github.com/rickb777/sqlgen2/parse/exit"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"strings"
	"time"
)

func main() {
	start := time.Now()

	var oFile, typeName, prefix, list, kind, tableName, tagsFile, genSetters string
	var flags = FuncFlags{}
	var pgx, all, read, create, gofmt, jsonFile, yamlFile, showVersion bool

	flag.StringVar(&oFile, "o", "", "Output file name; optional. Use '-' for stdout.\n"+
		"\tIf omitted, the first input filename is used with '_sql.go' suffix.")
	flag.StringVar(&typeName, "type", "", "The type to analyse; required.\n"+
		"\tThis is expressed in the form 'pkg.Name'")
	flag.StringVar(&prefix, "prefix", "", "Prefix for names of generated types; optional.\n"+
		"\tUse this if you need to avoid name collisions.")
	flag.StringVar(&list, "list", "", "List type for slice of model objects; optional.")
	flag.StringVar(&kind, "kind", "Table", "Kind of model: you could use 'Table', 'View', 'Join' etc as required.")
	flag.StringVar(&tableName, "table", "", "The name for the database table; default is based on the struct name as a plural.")
	flag.StringVar(&tagsFile, "tags", "", "A YAML file containing tags that augment and override any in the Go struct(s); optional.\n"+
		"\tTags control the SQL type, size, column name, indexes etc.")

	// filters for what gets generated
	flag.BoolVar(&pgx, "pgx", false, "Generates code for github.com/jackc/pgx.")
	flag.BoolVar(&all, "all", false, "Shorthand for '-schema -read -count -insert -update -upsert -delete -slice'; recommended.\n"+
		"\tThis does not affect -setters.")
	flag.BoolVar(&read, "read", false, "Alias for -select")
	flag.BoolVar(&create, "create", false, "Alias for -insert")
	flag.BoolVar(&flags.Schema, "schema", false, "Generate SQL schema create/drop methods.")
	flag.BoolVar(&flags.Insert, "insert", false, "Generate SQL insert (create) methods.")
	flag.BoolVar(&flags.Exec, "exec", false, "Generate Exec method. This is also provided with -update or -delete.")
	flag.BoolVar(&flags.Select, "select", false, "Generate SQL select (read) methods; also enables -count.")
	flag.BoolVar(&flags.Count, "count", false, "Generate SQL count methods.")
	flag.BoolVar(&flags.Update, "update", false, "Generate SQL update methods.")
	flag.BoolVar(&flags.Upsert, "upsert", false, "Generate SQL upsert methods; ignored if there is no primary key.")
	flag.BoolVar(&flags.Delete, "delete", false, "Generate SQL delete methods.")
	flag.BoolVar(&flags.Slice, "slice", false, "Generate SQL slice (column select) methods.")
	flag.BoolVar(&flags.Scan, "scan", false, "Generate exported row scan functions (these are normally unexported).")
	flag.StringVar(&genSetters, "setters", "none", "Generate setters for fields of your type (see -type): none, optional, exported, all.\n"+
		"\tFields that are pointers are assumed to be optional.")

	flag.BoolVar(&output.Verbose, "v", false, "Show progress messages.")
	flag.BoolVar(&parse.Debug, "z", false, "Show debug messages.")
	flag.BoolVar(&parse.PrintAST, "ast", false, "Trace the whole astract syntax tree (very verbose).")
	flag.BoolVar(&gofmt, "gofmt", false, "Format and simplify the generated code nicely.")
	flag.BoolVar(&jsonFile, "json", false, "Read/print the table description in JSON (overrides Go parsing if the JSON file exists).")
	flag.BoolVar(&yamlFile, "yaml", false, "Read/print the table description in YAML (overrides Go parsing if the YAML file exists).")
	flag.BoolVar(&showVersion, "version", false, "Show the version.")

	flag.Parse()

	if showVersion {
		fmt.Println(appVersion)
		os.Exit(0)
	}

	output.Require(flag.NArg() > 0, "At least one input file (or path) is required; put this after the other arguments.\n")

	if read {
		flags.Select = true
	}

	if flags.Select {
		flags.Count = true
	}

	if create {
		flags.Insert = true
	}

	if flags.Upsert {
		flags.Insert = true
		flags.Update = true
	}

	if all {
		flags = AllFuncFlags
	}

	output.Require(len(typeName) > 3, "-type is required. This must specify a type, qualified with its local package in the form 'pkg.Name'.\n", typeName)
	words := strings.Split(typeName, ".")
	output.Require(len(words) == 2, "type %q requires a package name prefix.\n", typeName)
	pkg, name := words[0], words[1]
	mainPkg := pkg

	if oFile == "" {
		oFile = flag.Args()[0]
		output.Require(strings.HasSuffix(oFile, ".go"), oFile+": must end '.go'")
		oFile = oFile[:len(oFile)-3] + "_sql.go"
		parse.DevInfo("oFile: %s\n", oFile)
	} else {
		mainPkg = LastDirName(oFile)
		parse.DevInfo("mainPkg: %s\n", mainPkg)
	}

	o := output.NewOutput(oFile)

	var table *schema.TableDescription

	if yamlFile {
		buf := &bytes.Buffer{}
		dec := yaml.NewDecoder(buf)
		table = readTableJson(o.Derive(".yml"), buf, dec)
		if table != nil {
			yamlFile = false
		}
	} else if jsonFile {
		buf := &bytes.Buffer{}
		dec := json.NewDecoder(buf)
		table = readTableJson(o.Derive(".json"), buf, dec)
		if table != nil {
			jsonFile = false
		}
	}

	if table == nil {
		output.Info("parsing %s\n", strings.Join(flag.Args(), ", "))

		// parse the Go source code file(s) to extract the required struct and return it as an AST.
		pkgStore, err := parse.Parse(flag.Args())
		output.Require(err == nil, "%v\n", err)
		//utter.Dump(pkgStore)

		tags, err := types.ReadTagsFile(tagsFile)
		if err != nil && !os.IsNotExist(err) {
			exit.Fail(1, "tags file %s failed: %s.\n", tagsFile, err)
		}

		// load the Tree into a schema Object
		table, err = Load(pkgStore, parse.LType{PkgName: pkg, Name: name}, mainPkg, tags)
		if err != nil {
			exit.Fail(1, "Go parser failed: %v.\n", err)
		}

		if parse.Debug {
			utter.Dump(table)
		}

		if len(table.Fields) < 1 {
			exit.Fail(1, "no fields found. Check earlier parser warnings.\n")
		}
	} else {
		output.Info("ignored %s\n", strings.Join(flag.Args(), ", "))
	}

	if yamlFile {
		buf := &bytes.Buffer{}
		enc := yaml.NewEncoder(buf)
		writeTableJson(o.Derive(".yml"), buf, enc, table)
	}

	if jsonFile {
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetIndent("", "  ")
		writeTableJson(o.Derive(".json"), buf, enc, table)
	}

	writeSqlGo(o, name, prefix, tableName, kind, list, mainPkg, genSetters, table, flags, pgx, gofmt)

	output.Info("%s took %v\n", o.Path(), time.Now().Sub(start))
}

func readTableJson(o output.Output, buf io.ReadWriter, dec decoder) *schema.TableDescription {
	err := o.ReadTo(buf)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		exit.Fail(1, "YAML reader from %s failed: %v.\n", o.Path(), err)
	}

	output.Info("reading %s\n", o.Path())
	table := schema.TableDescription{}
	err = dec.Decode(&table)
	if err != nil {
		exit.Fail(1, "YAML reader from %s failed: %v.\n", o.Path(), err)
	}
	return &table
}

func writeTableJson(o output.Output, buf io.ReadWriter, enc encoder, table *schema.TableDescription) {
	err := enc.Encode(table)
	if err != nil {
		exit.Fail(1, "YAML writer to %s failed: %v.\n", o.Path(), err)
	}

	err = o.Write(buf)
	if err != nil {
		exit.Fail(1, "YAML writer to %s failed: %v.\n", o.Path(), err)
	}
}

func writeSqlGo(o output.Output, name, prefix, tableName, kind, list, mainPkg, genSetters string, table *schema.TableDescription, flags FuncFlags, pgx, gofmt bool) {
	sql := "sql"
	api := "sqlapi"
	if pgx {
		sql = "pgx"
		api = "pgxapi"
	}
	view := NewView(name, prefix, tableName, list, sql, api)
	view.Table = table
	view.Thing = kind
	view.Interface1 = api + "." + PrimaryInterface(table, flags.Schema)
	if flags.Scan {
		view.Scan = "Scan"
	}

	setters := view.FilterSetters(genSetters)

	importSet := PackagesToImport(flags, pgx)

	ImportsForFields(table, importSet)
	ImportsForSetters(setters, importSet)

	buf := &bytes.Buffer{}

	WritePackageHeader(buf, mainPkg, appVersion)

	WriteImports(buf, importSet)

	WriteType(buf, view)

	WritePrimaryDeclarations(buf, view)

	if flags.Schema {
		WriteSchemaDeclarations(buf, view)
		WriteSchemaFunctions(buf, view)
	}

	if flags.Exec || flags.Update || flags.Delete {
		WriteExecFunc(buf, view)
	}

	WriteQueryRows(buf, view)
	WriteQueryThings(buf, view)
	WriteScanRows(buf, view)

	if flags.Select {
		WriteGetRow(buf, view)
		WriteSelectRowsFuncs(buf, view)
	}

	if flags.Count {
		WriteCountRowsFuncs(buf, view)
	}

	if flags.Slice {
		WriteSliceColumn(buf, view)
	}

	if flags.Insert {
		WriteConstructInsert(buf, view)
	}

	if flags.Update {
		WriteConstructUpdate(buf, view)
	}

	if flags.Insert {
		WriteInsertFunc(buf, view)
	}

	if flags.Update {
		WriteUpdateFunc(buf, view)
	}

	if flags.Upsert {
		WriteUpsertFunc(buf, view)
	}

	if flags.Delete {
		WriteDeleteFunc(buf, view)
	}

	WriteSetters(buf, view, setters)

	// formats the generated file using gofmt
	var pretty io.Reader = buf
	if gofmt {
		var err error
		pretty, err = Format(buf)
		output.Require(err == nil, "%s\n%v\n", string(buf.Bytes()), err)
	}

	o.Write(pretty)
}

type encoder interface {
	Encode(v interface{}) (err error)
}

type decoder interface {
	Decode(v interface{}) (err error)
}
