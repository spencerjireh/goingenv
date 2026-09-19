package diff

import (
	"reflect"
	"strings"
	"testing"
)

func side(m map[string]string) map[string][]byte {
	out := make(map[string][]byte, len(m))
	for k, v := range m {
		out[k] = []byte(v)
	}
	return out
}

func TestCompare(t *testing.T) {
	from := side(map[string]string{
		".env":         "A=1\nB=2\nC=3\n",
		"gone/.env":    "X=1\n",
		"same/.env":    "S=1\n",
		"reorder/.env": "P=1\nQ=2\n",
	})
	to := side(map[string]string{
		".env":         "A=1\nB=changed\nD=4\n",
		"new/.env":     "Y=1\nZ=2\n",
		"same/.env":    "S=1\n",
		"reorder/.env": "Q=2\nP=1\n",
	})

	r, err := Compare(from, to)
	if err != nil {
		t.Fatal(err)
	}
	want := Result{Files: []FileChange{
		{Path: ".env", Status: Modified, Keys: []KeyChange{{"B", Changed}, {"C", Removed}, {"D", Added}}},
		{Path: "gone/.env", Status: Removed, Keys: []KeyChange{{"X", Removed}}},
		{Path: "new/.env", Status: Added, Keys: []KeyChange{{"Y", Added}, {"Z", Added}}},
	}}
	if !reflect.DeepEqual(r, want) {
		t.Errorf("Compare =\n%+v\nwant\n%+v", r, want)
	}
	if !r.Changed() {
		t.Error("Changed() = false")
	}
}

func TestCompareEqualAndEmpty(t *testing.T) {
	r, err := Compare(side(map[string]string{".env": "A=1"}), side(map[string]string{".env": "A=1"}))
	if err != nil || r.Changed() {
		t.Errorf("equal sides: %+v, %v", r, err)
	}
	r, err = Compare(nil, nil)
	if err != nil || r.Changed() {
		t.Errorf("empty sides: %+v, %v", r, err)
	}
	r, err = Compare(side(map[string]string{".env": "A=1"}), nil)
	if err != nil || len(r.Files) != 1 || r.Files[0].Status != Removed {
		t.Errorf("against empty: %+v, %v", r, err)
	}
}

func TestCompareParseError(t *testing.T) {
	_, err := Compare(side(map[string]string{".env": `A="open`}), nil)
	if err == nil || !strings.Contains(err.Error(), ".env") {
		t.Errorf("err = %v", err)
	}
}

func TestFormatMasksValues(t *testing.T) {
	r, err := Compare(side(map[string]string{".env": "SECRET=old-value\n"}), side(map[string]string{".env": "SECRET=new-value\nNEW=fresh\n"}))
	if err != nil {
		t.Fatal(err)
	}
	out := Format(r)
	for _, leak := range []string{"old-value", "new-value", "fresh"} {
		if strings.Contains(out, leak) {
			t.Errorf("output leaks %q:\n%s", leak, out)
		}
	}
	for _, want := range []string{"~ .env (modified)", "~ SECRET", "+ NEW", "1 file(s) differ"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if Format(Result{}) != "No differences\n" {
		t.Errorf("empty format = %q", Format(Result{}))
	}
}
