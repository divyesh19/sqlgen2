package demo

// This example demonstrates
//   * no primary key column (which is unusual but permitted)
//   * an index that spans multiple columns
//   * metadata is written to/read from YAML file

//go:generate sqlgen -yaml -type demo.Compound -o compound_sql.go -gofmt -all -v -prefix Db category.go compound.go

type Compound struct {
	Alpha    string `sql:"unique: alpha_beta"`
	Beta     string `sql:"unique: alpha_beta"`
	Category Category
}
