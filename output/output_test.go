package output

import (
	"bytes"
	. "github.com/onsi/gomega"
	"io"
	"os"
	"testing"
)

func TestNewOutput(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := []struct {
		input, dirs, name, derived, pkg string
	}{
		{"", ".", "", "", ""},
		{"-", ".", "-", "-", ""},
		{"foo.go", ".", "foo.go", "foo.yml", ""},
		{"bar/foo.go", "bar", "foo.go", "foo.yml", "bar"},
		{"zob/bar/foo.go", "zob/bar", "foo.go", "foo.yml", "bar"},
	}
	for _, c := range cases {
		o := NewOutput(c.input)
		g.Expect(o.Dirs).To(Equal(c.dirs))
		g.Expect(o.Name).To(Equal(c.name))
		g.Expect(o.Pkg()).To(Equal(c.pkg))

		d := o.Derive(".yml")
		g.Expect(d.Name).To(Equal(c.derived))
	}
}

func TestWriteNoAction(t *testing.T) {
	o := NewOutput("")
	content := bytes.NewBufferString("some content")
	result := &bytes.Buffer{}
	o.Fallback = result

	o.Write(content)

	if result.String() != "" {
		t.Errorf("Got %q", result.String())
	}
}

func TestWriteStdout(t *testing.T) {
	o := NewOutput("-")
	content := bytes.NewBufferString("some content")
	result := &bytes.Buffer{}
	o.Fallback = result

	o.Write(content)

	if result.String() != "some content" {
		t.Errorf("Got %q", result.String())
	}
}

func TestWriteFile_createError(t *testing.T) {
	Os = &stubOs{
		createErr: os.ErrInvalid,
	}
	o := NewOutput("foo.go")

	_, err := o.create()

	if err == nil {
		t.Errorf("Expected an error")
	}
}

func TestWriteFile_simpleFileWrite(t *testing.T) {
	result := &bytes.Buffer{}
	stub := &stubOs{
		createFile: &nopCloser{result},
	}
	Os = stub

	o := NewOutput("foo.go")
	content := bytes.NewBufferString("some content")

	o.Write(content)

	if result.String() != "some content" {
		t.Errorf("Got %q", result.String())
	}
	if stub.createName != "./foo.go" {
		t.Errorf("Got %q", stub.createName)
	}
}

func TestWriteFile_createDirectoryAndFile(t *testing.T) {
	result := &bytes.Buffer{}
	stub := &stubOs{
		createFile: &nopCloser{result},
	}
	Os = stub

	o := NewOutput("bar/foo.go")
	content := bytes.NewBufferString("some content")

	o.Write(content)

	if result.String() != "some content" {
		t.Errorf("Got %q", result.String())
	}
	if stub.createName != "bar/foo.go" {
		t.Errorf("Got %q", stub.createName)
	}
	if stub.mkdirAllPath != "bar" {
		t.Errorf("Got %q", stub.mkdirAllPath)
	}
}

//-------------------------------------------------------------------------------------------------

type stubOs struct {
	createFile io.WriteCloser
	createErr  error
	createName string

	mkdirAllPath string
	mkdirAllErr  error
}

func (s *stubOs) Create(name string) (io.WriteCloser, error) {
	s.createName = name
	return s.createFile, s.createErr
}

func (s *stubOs) Open(name string) (io.ReadCloser, error) {
	return nil, nil
}

func (s *stubOs) MkdirAll(path string, perm os.FileMode) error {
	s.mkdirAllPath = path
	return s.mkdirAllErr
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
