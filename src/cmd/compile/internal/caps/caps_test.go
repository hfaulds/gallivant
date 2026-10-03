// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package caps

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want Cap
		ok   bool
	}{
		{"file.read", CapFileRead, true},
		{"file.write", CapFileWrite, true},
		{"net", CapNet, true},
		{"exec", CapExec, true},
		{"env", CapEnv, true},
		{"unsafe", CapUnsafe, true},
		{"  net  ", CapNet, true}, // whitespace is trimmed
		{"junk", "", false},
		{"", "", false},
		{"FILE:READ", "", false}, // case-sensitive
	}
	for _, c := range cases {
		got, ok := Parse(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("Parse(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestRequired(t *testing.T) {
	cases := []struct {
		pkg, name string
		want      []Cap
	}{
		{"os", "Open", []Cap{CapFileRead}},
		{"os", "OpenFile", []Cap{CapFileRead, CapFileWrite}},
		{"os", "WriteFile", []Cap{CapFileWrite}},
		{"os", "Getenv", []Cap{CapEnv}},
		{"io/ioutil", "ReadFile", []Cap{CapFileRead}},
		{"io/ioutil", "TempDir", []Cap{CapFileWrite}},
		{"net", "Dial", []Cap{CapNet}},
		{"net/http", "Get", []Cap{CapNet}},
		{"net/http", "Client.Do", []Cap{CapNet}},
		{"net/http", "ListenAndServe", []Cap{CapNet}},
		{"os", "Stat", []Cap{CapFileRead}},
		{"os", "Symlink", []Cap{CapFileWrite}},
		{"os", "StartProcess", []Cap{CapExec}},
		{"os", "Expand", []Cap{CapEnv}},
		{"os", "UserHomeDir", []Cap{CapEnv}},
		{"path/filepath", "Walk", []Cap{CapFileRead}},
		{"path/filepath", "Glob", []Cap{CapFileRead}},
		{"fmt", "Println", []Cap{CapFileWrite}},
		{"fmt", "Scan", []Cap{CapFileRead}},
		{"log", "Printf", []Cap{CapFileWrite}},
		{"net", "LookupMX", []Cap{CapNet}},
		{"net", "Dialer.Dial", []Cap{CapNet}},
		{"net", "Resolver.LookupHost", []Cap{CapNet}},
		{"net/http", "ServeFile", []Cap{CapNet, CapFileRead}},
		{"net/http", "ServeTLS", []Cap{CapNet}},
		{"crypto/tls", "Dial", []Cap{CapNet}},
		{"crypto/tls", "Listen", []Cap{CapNet}},
		{"net/smtp", "SendMail", []Cap{CapNet}},
		{"os/exec", "Command", []Cap{CapExec}},
		{"os/exec", "Cmd.Run", []Cap{CapExec}},
		{"syscall", "ForkExec", []Cap{CapExec}},
		{"syscall", "Socket", []Cap{CapNet}},
		{"syscall", "Read", []Cap{CapFileRead}},
		{"syscall", "Getenv", []Cap{CapEnv}},
		// Unknown symbols / packages return nil.
		{"os", "Unknown", nil},
		{"fmt", "Fprintf", nil},
		{"github.com/foo/bar", "Open", nil},
	}
	for _, c := range cases {
		got := Required(c.pkg, c.name)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Required(%q,%q) = %v, want %v", c.pkg, c.name, got, c.want)
		}
	}
}

func TestRequiredOnImport(t *testing.T) {
	cases := []struct {
		pkg  string
		want []Cap
	}{
		{"C", []Cap{CapUnsafe}},
		{"unsafe", []Cap{CapUnsafe}},
		{"plugin", []Cap{CapUnsafe}},
		{"reflect", []Cap{CapUnsafe}},
		{"os", nil},
		{"fmt", nil},
		{"net/http", nil},
		{"github.com/foo/bar", nil},
	}
	for _, c := range cases {
		got := RequiredOnImport(c.pkg)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("RequiredOnImport(%q) = %v, want %v", c.pkg, got, c.want)
		}
	}
}

func TestTablesReturnCopies(t *testing.T) {
	got := RequiredOnImport("unsafe")
	if len(got) == 0 {
		t.Fatal("expected non-empty slice")
	}
	got[0] = CapFileRead
	if again := RequiredOnImport("unsafe"); again[0] != CapUnsafe {
		t.Errorf("requiredOnImport table was mutated: %v", again)
	}

	got = Required("os", "Open")
	if len(got) == 0 {
		t.Fatal("expected non-empty slice")
	}
	got[0] = CapUnsafe
	if again := Required("os", "Open"); again[0] != CapFileRead {
		t.Errorf("required table was mutated: %v", again)
	}
}

func TestIsStdlib(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"", true},
		{"os", true},
		{"net/http", true},
		{"unsafe", true},
		{"C", true},
		{"vendor/golang.org/x/net/http2/hpack", true},
		{"github.com/foo/bar", false},
		{"example.com", false},
		{"golang.org/x/tools/go/analysis", false},
		{"gopkg.in/yaml.v3", false},
	}
	for _, c := range cases {
		if got := IsStdlib(c.path); got != c.want {
			t.Errorf("IsStdlib(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestSetSortedIsStable(t *testing.T) {
	s := NewSet(CapEnv, CapFileRead, CapNet, CapExec, CapFileWrite)
	got := s.Sorted()
	want := []Cap{CapFileRead, CapFileWrite, CapNet, CapExec, CapEnv}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Sorted() = %v, want %v", got, want)
	}
}

func TestSetHelpers(t *testing.T) {
	a := NewSet(CapFileRead, CapNet)
	b := NewSet(CapNet, CapExec)

	if !a.Has(CapFileRead) || a.Has(CapExec) {
		t.Errorf("Has wrong: a=%v", a)
	}

	a.Add(CapEnv)
	if !a.Has(CapEnv) {
		t.Errorf("Add did not insert")
	}

	u := a.Union(b)
	wantUnion := []Cap{CapFileRead, CapNet, CapExec, CapEnv}
	if got := u.Sorted(); !reflect.DeepEqual(got, wantUnion) {
		t.Errorf("Union sorted = %v, want %v", got, wantUnion)
	}
	if a.Has(CapExec) || b.Has(CapFileRead) {
		t.Errorf("Union mutated a receiver")
	}

	d := a.Diff(b)
	wantDiff := []Cap{CapFileRead, CapEnv}
	if got := d.Sorted(); !reflect.DeepEqual(got, wantDiff) {
		t.Errorf("Diff sorted = %v, want %v", got, wantDiff)
	}
}

func TestAllOrdering(t *testing.T) {
	want := []Cap{CapFileRead, CapFileWrite, CapNet, CapExec, CapEnv, CapUnsafe}
	if got := All(); !reflect.DeepEqual(got, want) {
		t.Errorf("All() = %v, want %v", got, want)
	}
}

func TestFormatList(t *testing.T) {
	if got := FormatList(nil); got != "[]" {
		t.Errorf("FormatList(nil) = %q", got)
	}
	if got := FormatList([]Cap{CapFileWrite, CapNet}); got != "[file.write, net]" {
		t.Errorf("FormatList = %q", got)
	}
}

// codec is an in-memory Encoder/Decoder pair for testing Fact round trips.
type codec struct {
	items []any
}

func (c *codec) Bool(b bool) bool { c.items = append(c.items, b); return b }
func (c *codec) Len(n int)        { c.items = append(c.items, n) }
func (c *codec) String(s string)  { c.items = append(c.items, s) }

type decoder struct {
	items []any
	i     int
}

func (d *decoder) next() any  { v := d.items[d.i]; d.i++; return v }
func (d *decoder) Bool() bool { return d.next().(bool) }
func (d *decoder) Len() int   { return d.next().(int) }
func (d *decoder) String() string {
	return d.next().(string)
}

func TestFactRoundTrip(t *testing.T) {
	in := &Fact{
		Caps:       []Cap{CapFileWrite, CapNet},
		ModulePath: "example.com/mod",
		Chain: map[string]string{
			"net":        "example.com/mod/a.go:12: net.Dial",
			"file.write": "example.com/mod/a.go:7: import \"example.com/dep\" → example.com/dep/b.go:3: os.Create",
		},
	}
	var c codec
	in.Write(&c)
	d := &decoder{items: c.items}
	out := Read(d)
	if d.i != len(c.items) {
		t.Errorf("Read consumed %d of %d items", d.i, len(c.items))
	}
	if !reflect.DeepEqual(in, out) {
		t.Errorf("round trip mismatch:\n in: %+v\nout: %+v", in, out)
	}

	// nil fact
	c = codec{}
	(*Fact)(nil).Write(&c)
	d = &decoder{items: c.items}
	if out := Read(d); out != nil {
		t.Errorf("Read(nil fact) = %+v, want nil", out)
	}
	if d.i != len(c.items) {
		t.Errorf("Read consumed %d of %d items for nil fact", d.i, len(c.items))
	}

	// empty fact
	c = codec{}
	(&Fact{}).Write(&c)
	d = &decoder{items: c.items}
	out = Read(d)
	if out == nil || len(out.Caps) != 0 || out.Chain != nil || out.ModulePath != "" {
		t.Errorf("Read(empty fact) = %+v", out)
	}
}
