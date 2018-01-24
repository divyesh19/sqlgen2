package parse

import (
	"io/ioutil"
	"os"
	"testing"
	. "github.com/onsi/gomega"
)

func TestParseTag(t *testing.T) {
	RegisterTestingT(t)

	tagTests := []struct {
		raw string
		tag *Tag
	}{
		{
			TagKey + `:"-"`,
			&Tag{Skip: true},
		},
		{
			TagKey + `:"prefixed: true"`,
			&Tag{Prefixed: true},
		},
		{
			TagKey + `:"pk: true"`,
			&Tag{Primary: true, Auto: false},
		},
		{
			TagKey + `:"pk: true, auto: true"`,
			&Tag{Primary: true, Auto: true},
		},
		{
			TagKey + `:"auto: true"`,
			&Tag{Primary: false, Auto: true},
		},
		{
			TagKey + `:"name: foo"`,
			&Tag{Name: "foo"},
		},
		{
			TagKey + `:"type: varchar"`,
			&Tag{Type: "varchar"},
		},
		{
			TagKey + `:"size: 2048"`,
			&Tag{Size: 2048},
		},
		{
			TagKey + `:"index: fake_index"`,
			&Tag{Index: "fake_index"},
		},
		{
			TagKey + `:"unique: fake_unique_index"`,
			&Tag{Unique: "fake_unique_index"},
		},
		{
			TagKey + `:"fk: alpha.ID, onupdate: setnull, ondelete: setdefault"`,
			&Tag{ForeignKey: "alpha.ID", OnUpdate: "set null", OnDelete: "set default"},
		},
		{
			TagKey + `:"fk: alpha.ID, onupdate: 'set null', ondelete: 'set default'"`,
			&Tag{ForeignKey: "alpha.ID", OnUpdate: "set null", OnDelete: "set default"},
		},
	}

	for _, test := range tagTests {
		got, err := ParseTag(test.raw)
		Ω(err).Should(BeNil(), test.raw)
		Ω(got).Should(Equal(test.tag), test.raw)
	}
}

func TestParseValidation(t *testing.T) {
	RegisterTestingT(t)

	tagTests := []struct {
		raw string
		err string
	}{
		{
			TagKey + `:"encode: x"`,
			`unrecognised encode value "x"`,
		},
		{
			TagKey + `:"fk: x"`,
			`fk value ("x") must be in 'tablename.column' form`,
		},
		{
			TagKey + `:"onupdate: x"`,
			`unrecognised onupdate value "x"`,
		},
		{
			TagKey + `:"ondelete: x"`,
			`unrecognised ondelete value "x"`,
		},
		{
			TagKey + `:"onupdate: x, ondelete: y"`,
			`unrecognised onupdate value "x"; unrecognised ondelete value "y"`,
		},
		{
			TagKey + `:"size: -1"`,
			`size cannot be negative (-1)`,
		},
	}

	for _, test := range tagTests {
		_, err := ParseTag(test.raw)
		Ω(err).Should(Not(BeNil()), test.raw)
		Ω(err.Error()).Should(Equal(test.err), test.raw)
	}
}

func TestReadTagsFile(t *testing.T) {
	RegisterTestingT(t)

	file := os.TempDir() + "/sqlgen2-test.yaml"
	defer os.Remove(file)

	yml := `
Id:
  pk: true
  auto: true

Foo:
  name: fooish
  type: blob
`

	err := ioutil.WriteFile(file, []byte(yml), 0644)
	Ω(err).Should(BeNil())

	tags, err := ReadTagsFile(file)
	Ω(err).Should(BeNil())
	Ω(len(tags)).Should(Equal(2))

	id := tags["Id"]
	Ω(id).Should(Equal(Tag{Primary: true, Auto: true}))

	foo := tags["Foo"]
	Ω(foo).Should(Equal(Tag{Name: "fooish", Type: "blob"}))
}
