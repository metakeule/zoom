package zoom

/*
import (
	"bytes"
	"testing"
)

func TestAddIndex(t *testing.T) {
	idx := NewIndex("", "", 255, 255, 1)

	tests := []struct {
		first string
		last  string
	}{
		{"Donald", "Duck"},
		{"Mickey", "Mouse"},
	}

	var bf bytes.Buffer
	var err error

	for _, test := range tests {
		err = idx.add(test.first, test.last, &bf)
		if err != nil {
			t.Errorf("can't add %s %s: %s", test.first, test.last, err.Error())
		}
	}

	vals := bf.Bytes()
	var res string

	for _, test := range tests {
		res, err = idx.find(test.first, bytes.NewReader(vals))

		if err != nil {
			t.Errorf("can't find %s: %s", test.first, err.Error())
		}

		if res != test.last {
			t.Errorf("can't find %s: expected: %s, got: %s", test.first, test.last, res)
		}

	}

}

func TestRemoveIndex(t *testing.T) {
	idx := NewIndex("", "", 255, 255, 1)

	tests := []struct {
		first string
		last  string
	}{
		{"Donald", "Duck"},
		{"Mickey", "Mouse"},
	}

	var bf bytes.Buffer
	var err error

	for _, test := range tests {
		err = idx.add(test.first, test.last, &bf)
		if err != nil {
			t.Errorf("can't add %s %s: %s", test.first, test.last, err.Error())
		}
	}

	vals := bf.Bytes()
	var res string

	for _, test := range tests {
		res, err = idx.find(test.first, bytes.NewReader(vals))

		if err != nil {
			t.Errorf("can't find %s: %s", test.first, err.Error())
		}

		if res != test.last {
			t.Errorf("can't find %s: expected: %s, got: %s", test.first, test.last, res)
		}

	}

}
*/
