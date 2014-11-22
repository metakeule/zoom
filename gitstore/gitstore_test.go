package gitstore

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/metakeule/zoom"
)

func withGit(fn func(*Git)) error {
	dir, err := ioutil.TempDir(os.TempDir(), "gitstore_")

	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	git, err := Open(dir)

	if err != nil {
		return err
	}

	fn(git)
	return nil
}

func TestNodes(t *testing.T) {
	tests := [...]map[string]interface{}{
		{
			"Age":       float64(44),
			"FirstName": "Donald",
			"LastName":  "Duck",
		},
		{
			"Age":       float64(42),
			"FirstName": "Daisy",
			"LastName":  "Duck",
		},
	}

	testsIds := make([]string, len(tests))

	err := withGit(func(git *Git) {
		for i, test := range tests {
			var n = zoom.NewNode("shard1")
			n.Set(test)
			_, err := git.Transaction("save test", n.Save)

			if err != nil {
				t.Fatal(err)
			}
			testsIds[i] = n.ID()
		}

		for i, id := range testsIds {
			var n = zoom.NewNode(id)

			_, err := git.Transaction("get", func(st zoom.Store) error {
				query := []string{}

				for k := range tests[i] {
					query = append(query, k)
				}

				return n.LoadProperties(st, query)
			})

			if err != nil {
				t.Fatal(err)
			}
			props := n.Properties()

			for k, v := range props {
				if tests[i][k] != v {
					t.Errorf("test[%d][%#v] = %#v, expected %#v", i, k, v, tests[i][k])
				}
			}

		}
	})

	if err != nil {
		t.Fatal(err)
	}

}

func TestRelation(t *testing.T) {
	// maps from name to name
	tests := [...]map[string]string{
		{
			"A": "B",
		},
		{
			"C": "D",
		},
	}

	testsIds := make([]string, len(tests))

	nameProp := "Name"
	pointsToProp := "points-to"

	err := withGit(func(git *Git) {

		for i, pair := range tests {
			if len(pair) != 1 {
				t.Fatalf("test map should only have one entry, but has %#v", pair)
			}

			for namefrom, nameTo := range pair {

				var from = zoom.NewNode("shard1")
				from.Set(map[string]interface{}{nameProp: namefrom})

				var to = zoom.NewNode("shard1")
				to.Set(map[string]interface{}{nameProp: nameTo})

				from.Update(pointsToProp, to)

				_, err := git.Transaction("save test", to.Save, from.Save)

				if err != nil {
					t.Fatal(err)
				}
				testsIds[i] = from.ID()

			}

		}

		for i, id := range testsIds {
			var from = zoom.NewNode(id)
			var to *zoom.Node

			_, err := git.Transaction("get", func(st zoom.Store) error {
				query := []string{}

				for k := range tests[i] {
					query = append(query, k)
				}

				if err := from.LoadProperties(st, []string{nameProp}); err != nil {
					return err
				}

				if err := from.LoadRelations(st, map[string][]string{pointsToProp: []string{nameProp}}); err != nil {
					return err
				}

				to = from.Relation(pointsToProp)
				if err := to.LoadProperties(st, []string{nameProp}); err != nil {
					return err
				}

				return nil
			})

			if err != nil {
				t.Fatal(err)
			}
			fromName := from.Properties()[nameProp].(string)
			toName := to.Properties()[nameProp].(string)

			if tests[i][fromName] != toName {
				t.Errorf("tests[%d][%#v] = %#v, expected %#v", i, fromName, toName, tests[i][fromName])
			}
		}
	})

	if err != nil {
		t.Fatal(err)
	}

}
