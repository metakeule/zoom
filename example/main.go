package main

import (
	"fmt"
	"os"

	"github.com/metakeule/config"
	"github.com/metakeule/zoom"
	"github.com/metakeule/zoom/gitstore"
)

var base = "BASE"

func main() {
	cfg := config.New("ex", "0.1", []config.Option{
		{
			Name:     base,
			Help:     "top level directory of the gitdb",
			Type:     "string",
			Required: true,
		},
	})
	cfg.Load("ex is an example for the git library")

	store, err := gitstore.Open(cfg.GetString(base))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}

	// store.Git.Debug = true

	actions := []func(zoom.Store) error{}

	for i := 0; i < 100; i++ {
		fn := func() []func(zoom.Store) error {
			nadja := zoom.NewNode("shard1")

			nadja.Set(map[string]interface{}{
				"Age":       44,
				"FirstName": "Nadja",
				"LastName":  "Poetschki",
			})

			benny := zoom.NewNode("shard1")

			benny.Set(map[string]interface{}{
				"Age":       42,
				"FirstName": "Benny",
				"LastName":  "Arns",
			})

			nadja.Update("friend-of", benny)

			return []func(zoom.Store) error{nadja.Save, benny.Save}
		}

		actions = append(actions, fn()...)

	}

	_, err = store.Transaction("add persons and relations", actions...)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}
