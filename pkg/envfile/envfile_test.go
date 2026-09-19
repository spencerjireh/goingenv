package envfile

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want map[string]string
		keys []string
	}{
		{"simple", "A=1\nB=two\n", map[string]string{"A": "1", "B": "two"}, []string{"A", "B"}},
		{"comments and blanks", "# c\n\nA=1\n  # indented\nB=2", map[string]string{"A": "1", "B": "2"}, []string{"A", "B"}},
		{"export prefix", "export A=1\nexport  B=2\n", map[string]string{"A": "1", "B": "2"}, []string{"A", "B"}},
		{"double quotes with escapes", `A="x\ny\t\"q\" \\"`, map[string]string{"A": "x\ny\t\"q\" \\"}, []string{"A"}},
		{"single quotes literal", `A='x\ny $B'`, map[string]string{"A": `x\ny $B`}, []string{"A"}},
		{"multiline double", "A=\"line1\nline2\"\nB=2", map[string]string{"A": "line1\nline2", "B": "2"}, []string{"A", "B"}},
		{"multiline single", "A='l1\nl2'\nB=2", map[string]string{"A": "l1\nl2", "B": "2"}, []string{"A", "B"}},
		{"inline comment unquoted", "A=value # note\nB=a#b", map[string]string{"A": "value", "B": "a#b"}, []string{"A", "B"}},
		{"quoted hash kept", `A="v # not a comment"`, map[string]string{"A": "v # not a comment"}, []string{"A"}},
		{"no expansion", "A=${HOME}/x\nB=$A", map[string]string{"A": "${HOME}/x", "B": "$A"}, []string{"A", "B"}},
		{"empty value", "A=\nB=''", map[string]string{"A": "", "B": ""}, []string{"A", "B"}},
		{"duplicate keys", "A=1\nA=2\nB=3", map[string]string{"A": "2", "B": "3"}, []string{"A", "B"}},
		{"invalid lines skipped", "just prose\n1BAD=x\nA-B=y\nOK=1", map[string]string{"OK": "1"}, []string{"OK"}},
		{"crlf", "A=1\r\nB=2\r\n", map[string]string{"A": "1", "B": "2"}, []string{"A", "B"}},
		{"spaces around equals", "A = 1", map[string]string{"A": "1"}, []string{"A"}},
		{"empty file", "", map[string]string{}, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := Parse([]byte(tc.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := f.Map(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Map() = %#v, want %#v", got, tc.want)
			}
			if got := f.Keys(); !reflect.DeepEqual(got, tc.keys) {
				t.Errorf("Keys() = %#v, want %#v", got, tc.keys)
			}
		})
	}
}

func TestParseUnterminatedQuote(t *testing.T) {
	for _, in := range []string{`A="open`, "A='open\nB=2"} {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("Parse(%q) = nil error, want unterminated", in)
		}
	}
}

func TestKeysNeverNil(t *testing.T) {
	f, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.Keys() == nil {
		t.Error("Keys() returned nil")
	}
}

func TestEntryLines(t *testing.T) {
	f, err := Parse([]byte("# c\nA=1\n\nB=\"x\ny\"\nC=3"))
	if err != nil {
		t.Fatal(err)
	}
	want := []int{2, 4, 6}
	for i, e := range f.Entries {
		if e.Line != want[i] {
			t.Errorf("entry %d line = %d, want %d", i, e.Line, want[i])
		}
	}
}
