package gitstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/metakeule/gitlib"
	"github.com/metakeule/zoom"
	// "gopkg.in/vmihailenco/msgpack.v1"
)

type Git struct {
	*gitlib.Git
}

func Open(baseDir string) (*Git, error) {
	git, err := gitlib.NewGit(baseDir)
	if err != nil {
		return nil, err
	}
	if !git.IsInitialized() {
		// fmt.Println("initializing")
		err := git.Transaction(func(tx *gitlib.Transaction) error {
			if err := tx.InitWithReadme(strings.NewReader("first commit")); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return &Git{git}, nil
}

func (g *Git) Transaction(comment string, actions ...func(zoom.Store) error) (rolledback bool, err error) {
	err = g.Git.Transaction(func(tx *gitlib.Transaction) (err2 error) {
		if err2 = tx.InitWithReadme(strings.NewReader("create zoomDB")); err2 != nil {
			return
		}

		var store zoom.Store = &Store{tx}

		rolledback, err2 = zoom.Transaction(store, comment, actions...)
		return
	})
	return
}

type Store struct {
	*gitlib.Transaction
}

func nodeUUIDProp2path(uuid string) string {
	return fmt.Sprintf("node/props/%s/%s", uuid[:2], uuid[2:])
}

func nodeUUIDRel2path(uuid string) string {
	return fmt.Sprintf("node/rels/%s/%s", uuid[:2], uuid[2:])
}

func nodeUUIDPool2path(uuid string) string {
	return fmt.Sprintf("node/pools/%s/%s", uuid[:2], uuid[2:])
}

func (g *Store) Commit(comment string) error {
	treeSha, err := g.Transaction.WriteTree()
	if err != nil {
		return err
	}

	var parent string
	parent, err = g.ShowHeadsRef("master")
	if err != nil {
		return err
	}

	var commitSha string
	commitSha, err = g.CommitTree(treeSha, parent, strings.NewReader(comment))

	if err != nil {
		return err
	}

	return g.UpdateHeadsRef("master", commitSha)
}

func (g *Store) save(path string, isNew bool, data interface{}) error {
	var buf bytes.Buffer
	// enc := msgpack.NewEncoder(&buf)
	enc := json.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return err
	}

	var sha1 string
	sha1, err = g.Transaction.WriteHashObject(&buf)
	if err != nil {
		return err
	}

	if isNew {
		err = g.Transaction.AddIndexCache(sha1, path)
	} else {
		err = g.Transaction.UpdateIndexCache(sha1, path)
	}
	if err != nil {
		return err
	}
	return nil
}

func (g *Store) load(path string, data interface{}) error {
	var buf bytes.Buffer
	err := g.Transaction.ReadCatHeadFile(path, &buf)
	if err != nil {
		return err
	}
	//dec := msgpack.NewDecoder(&buf)
	dec := json.NewDecoder(&buf)
	return dec.Decode(data)
}

// only the props that have a key set are going to be changed
func (g *Store) SaveNodeProperties(nodeUuid string, isNew bool, props map[string]interface{}) error {
	path := nodeUUIDProp2path(nodeUuid)
	/*
		if !isNew {
			orig := map[string]interface{}{}
			err := g.load(path, &orig)
			if err != nil {
				return err
			}

			for k, v := range props {
				if v == nil {
					delete(orig, k)
				} else {
					orig[k] = v
				}
			}
			props = orig
		}
	*/
	return g.save(path, isNew, props)
}

// map relname => nodeUuid, only the relations that have a key set are going to be changed
func (g *Store) SaveNodeRelations(nodeUuid string, isNew bool, relations map[string]string) error {
	path := nodeUUIDRel2path(nodeUuid)
	/*
		if !isNew {
			orig := map[string]string{}
			err := g.load(path, &orig)
			if err != nil {
				return err
			}

			for k, v := range relations {
				if v == "" {
					delete(orig, k)
				} else {
					orig[k] = v
				}
			}
			relations = orig
		}
	*/
	return g.save(path, isNew, relations)
}

// TODO map poolname => []nodeUuid, only the pools that have a key set are going to be changed
func (g *Store) SaveNodePools(nodeUuid string, isNew bool, pools map[string][]string) error {
	path := nodeUUIDPool2path(nodeUuid)
	/*
		if !isNew {
			orig := map[string][]string{}
			err := g.load(path, &orig)
			if err != nil {
				return err
			}

			for k, v := range pools {
				if len(v) == 0 {
					delete(orig, k)
				} else {
					orig[k] = v
				}
			}
			pools = orig
		}
	*/
	return g.save(path, isNew, pools)
}

// TODO Rollback any actions that have been taken since the last commit
// stage should be cleared and any newly added data should be removed
func (g *Store) Rollback() error {
	return nil
}

// TODO what happens on errors? changes will not be committed!
func (g *Store) RemoveNode(nodeUuid string) error {
	paths := []string{
		nodeUUIDProp2path(nodeUuid),
		nodeUUIDRel2path(nodeUuid),
		nodeUUIDPool2path(nodeUuid),
	}

	for _, path := range paths {
		err := g.Transaction.RmIndex(path)
		if err != nil {
			return err
		}
	}
	return nil
}

// only the properties that exist make it into the returned map
// it is no error if a requested property does not exist for a node
// the caller has to check the returned map against the requested props if
// she wants to check, if all requested properties have been returned
func (g *Store) GetNodeProperties(nodeUuid string, requestedProps []string) (props map[string]interface{}, err error) {
	path := nodeUUIDProp2path(nodeUuid)
	orig := map[string]interface{}{}
	err = g.load(path, &orig)
	if err != nil {
		return nil, err
	}
	props = map[string]interface{}{}

	for _, req := range requestedProps {
		v, has := orig[req]
		if has {
			props[req] = v
		}
	}
	return
}

// the returned map has as values the uuids of the nodes
// only the relations that exist make it into the returned map
// it is no error if a requested relation does not exist for a node
// the caller has to check the returned map against the requested rels if
// she wants to check, if all requested relations have been returned
// also there is no guarantee that the nodes which uuids are returned do still exist.
// there must be wrappers put around the store to ensure this (preferably by using indices)
func (g *Store) GetNodeRelations(nodeUuid string, requestedRels []string) (rels map[string]string, err error) {
	path := nodeUUIDRel2path(nodeUuid)
	orig := map[string]string{}
	err = g.load(path, &orig)
	if err != nil {
		return nil, err
	}
	rels = map[string]string{}

	for _, req := range requestedRels {
		v, has := orig[req]
		if has {
			rels[req] = v
		}
	}
	return
}

// the returned map has as values slices of uuids of the nodes
// only the pools that exist make it into the returned map
// it is no error if a requested pool does not exist for a node
// the caller has to check the returned map against the requested pools if
// she wants to check, if all requested pools have been returned
// also there is no guarantee that the nodes which uuids are returned do still exist.
// there must be wrappers put around the store to ensure this (preferably by using indices)
func (g *Store) GetNodePools(nodeUuid string, requestedPools []string) (pools map[string][]string, err error) {
	path := nodeUUIDPool2path(nodeUuid)
	orig := map[string][]string{}
	err = g.load(path, &orig)
	if err != nil {
		return nil, err
	}
	pools = map[string][]string{}

	for _, req := range requestedPools {
		v, has := orig[req]
		if has {
			pools[req] = v
		}
	}
	return
}
