package zoom_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/metakeule/zoom"
	"github.com/metakeule/zoom/gitstore"
)

var base = "BASE"

func TestStore(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "zoom_")

	if err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("temp dir: %s", dir)

	defer os.RemoveAll(dir)

	store, err := gitstore.Open(dir)
	// store.Debug = true

	if err != nil {
		t.Fatal(err)
	}

	// store.Git.Debug = true

	actions := []func(zoom.Store) error{}

	nadja := zoom.NewNode("shard1")
	benny := zoom.NewNode("shard1")

	fn := func() []func(zoom.Store) error {

		nadja.Set(map[string]interface{}{
			"Age":       44,
			"FirstName": "Nadja",
			"LastName":  "Poetschki",
		})

		benny.Set(map[string]interface{}{
			"Age":       42,
			"FirstName": "Benny",
			"LastName":  "Arns",
		})

		nadja.Update("friend-of", benny)

		return []func(zoom.Store) error{nadja.Save, benny.Save}
	}

	actions = append(actions, fn()...)

	_, err = store.Transaction("add persons and relations", actions...)

	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(nadja.ID())

	var nadja2 = zoom.NewNode(nadja.ID())
	var benny2 = zoom.NewNode(benny.ID())
	_, err = store.Transaction("get", func(st zoom.Store) error {
		if err := nadja2.LoadProperties(st, []string{"Age", "FirstName", "LastName"}); err != nil {
			return err
		}

		if err := benny2.LoadProperties(st, []string{"Age", "FirstName", "LastName"}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	propsNadja := nadja2.Properties()

	ageNadja, ok := propsNadja["Age"].(float64)

	if !ok || ageNadja != 44 {
		t.Errorf("wrong age for nadja expected: %v, got %v", 44, ageNadja)
	}

	firstNameNadja, ok := propsNadja["FirstName"].(string)

	if !ok || firstNameNadja != "Nadja" {
		t.Errorf("wrong FirstName for nadja expected: %v, got %v", "Nadja", firstNameNadja)
	}

	propsBenny := benny2.Properties()

	ageBenny, ok := propsBenny["Age"].(float64)

	if !ok || ageBenny != 42 {
		t.Errorf("wrong age for nadja expected: %v, got %v", 42, ageBenny)
	}

	firstNameBenny, ok := propsBenny["FirstName"].(string)

	if !ok || firstNameBenny != "Benny" {
		t.Errorf("wrong FirstName for nadja expected: %v, got %v", "Benny", firstNameBenny)
	}

}
