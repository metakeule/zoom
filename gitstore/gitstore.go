package gitstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

// map relname => nodeUuid, only the texts that have a key set are going to be changed
func (s *Store) SaveNodeTexts(uuid string, shard string, isNew bool, texts map[string]string) error {

	for textPath, text := range texts {
		path := s.textPath(shard, uuid, textPath)
		rd := strings.NewReader(text)
		sha1, err := s.Transaction.WriteHashObject(rd)
		if err != nil {
			return err
		}

		if isNew {
			err = s.Transaction.AddIndexCache(sha1, path)
		} else {
			err = s.Transaction.UpdateIndexCache(sha1, path)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// map poolname => []nodeUuid, only the blobs that have a key set are going to be changed
func (s *Store) SaveNodeBlobs(uuid string, shard string, isNew bool, blobs map[string]io.Reader) error {

	for blobPath, blob := range blobs {
		path := s.blobPath(shard, uuid, blobPath)

		sha1, err := s.Transaction.WriteHashObject(blob)
		if err != nil {
			return err
		}

		if isNew {
			err = s.Transaction.AddIndexCache(sha1, path)
		} else {
			err = s.Transaction.UpdateIndexCache(sha1, path)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetNodeBlobs(uuid string, shard string, requestedBlobs []string) (blobs map[string]io.Reader, err error) {
	blobs = map[string]io.Reader{}

	for _, blob := range requestedBlobs {
		var buf bytes.Buffer
		err = s.ReadCatHeadFile(s.blobPath(shard, uuid, blob), &buf)
		if err != nil {
			return
		}
		blobs[blob] = &buf
	}
	return

}

func (s *Store) GetNodeTexts(uuid string, shard string, requestedTexts []string) (
	texts map[string]string, err error) {
	texts = map[string]string{}
	for _, text := range requestedTexts {
		var buf bytes.Buffer
		err = s.ReadCatHeadFile(s.textPath(shard, uuid, text), &buf)
		if err != nil {
			return
		}
		texts[text] = buf.String()
	}
	return
}

func (s *Store) SaveEdges(category, shard, uuid string, isNew bool, edges map[string]string) error {
	path := s.edgePath(category, shard, uuid)
	return s.save(path, isNew, edges)
}

// RemoveEdges also removes the properties node of an edge
func (s *Store) RemoveEdges(category, shard, uuid string) error {
	edges, err := s.GetEdges(category, shard, uuid)
	if err != nil {
		return err
	}

	for _, propNode := range edges {
		shard, uuid, err := zoom.SplitID(propNode)
		if err != nil {
			return err
		}

		if err := s.RemoveNode(uuid, shard); err != nil {
			return err
		}
	}

	path := s.edgePath(category, shard, uuid)
	return s.RmIndex(path)
}

func (s *Store) GetEdges(category, shard, uuid string) (edges map[string]string, err error) {
	path := s.edgePath(category, shard, uuid)
	edges = map[string]string{}
	err = s.load(path, edges)
	return
}

func (s *Store) edgePath(category string, shard string, uuid string) string {
	//return fmt.Sprintf("node/props/%s/%s", uuid[:2], uuid[2:])
	return fmt.Sprintf("refs/%s/%s/%s/%s", category, shard, uuid[:2], uuid[2:])
}

func (s *Store) propPath(shard string, uuid string) string {
	//return fmt.Sprintf("node/props/%s/%s", uuid[:2], uuid[2:])
	return fmt.Sprintf("node/%s/%s/%s", shard, uuid[:2], uuid[2:])
}

func (s *Store) textPath(shard string, uuid string, key string) string {
	// return fmt.Sprintf("node/rels/%s/%s", uuid[:2], uuid[2:])
	return fmt.Sprintf("text/%s/%s/%s/%s", shard, uuid[:2], uuid[2:], key)
}

func (s *Store) blobPath(shard string, uuid string, blobpath string) string {
	return fmt.Sprintf("blob/%s/%s/%s/%s", shard, uuid[:2], uuid[2:], blobpath)
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
func (g *Store) SaveNodeProperties(uuid string, shard string, isNew bool, props map[string]interface{}) error {
	path := g.propPath(shard, uuid)
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
/*
func (g *Store) SaveNodeRelations(nodeUuid string, isNew bool, relations map[string]string) error {
	path := nodeUUIDRel2path(nodeUuid)
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
	return g.save(path, isNew, relations)
}
*/

// TODO map poolname => []nodeUuid, only the pools that have a key set are going to be changed
/*
func (g *Store) SaveNodePools(nodeUuid string, isNew bool, pools map[string][]string) error {
	path := nodeUUIDPool2path(nodeUuid)
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
	return g.save(path, isNew, pools)
}
*/

// TODO Rollback any actions that have been taken since the last commit
// stage should be cleared and any newly added data should be removed
func (g *Store) Rollback() error {
	return nil
}

/*
func (s *Store) edgePath(category string, shard string, uuid string) string {
	//return fmt.Sprintf("node/props/%s/%s", uuid[:2], uuid[2:])
	return fmt.Sprintf("refs/%s/%s/%s/%s", category, shard, uuid[:2], uuid[2:])
}
*/

// TODO what happens on errors? changes will not be committed!
// TODO what about the edges? must delete them all (and identify them all)
// to identify them we need something like
// ls refs/*/shard/uuid[:2], uuid[2:]
// return fmt.Sprintf("refs/%s/%s/%s/%s", category, shard, uuid[:2], uuid[2:])
func (g *Store) RemoveNode(uuid string, shard string) error {
	paths := []string{
		g.propPath(shard, uuid),
		fmt.Sprintf("text/%s/%s/%s", shard, uuid[:2], uuid[2:]),
		fmt.Sprintf("blob/%s/%s/%s", shard, uuid[:2], uuid[2:]),
	}

	files, err := g.LsFiles(fmt.Sprintf("refs/*/%s/%s/%s", shard, uuid[:2], uuid[2:]))
	if err != nil {
		return err
	}

	for _, file := range files {
		err := g.Transaction.RmIndex(file)
		if err != nil {
			return err
		}
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
func (g *Store) GetNodeProperties(uuid string, shard string, requestedProps []string) (props map[string]interface{}, err error) {
	path := g.propPath(shard, uuid)
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
/*
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
*/

// the returned map has as values slices of uuids of the nodes
// only the pools that exist make it into the returned map
// it is no error if a requested pool does not exist for a node
// the caller has to check the returned map against the requested pools if
// she wants to check, if all requested pools have been returned
// also there is no guarantee that the nodes which uuids are returned do still exist.
// there must be wrappers put around the store to ensure this (preferably by using indices)
/*
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
*/
