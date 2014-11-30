package gitstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/metakeule/gitlib"
	"github.com/metakeule/zoom"
	// "gopkg.in/vmihailenco/msgpack.v1"
)

/*
TODO implement sharding, i.e. add a layer on top, fullfilling the zoom.Store interface
and saving on the correspondig shard. (and add synchronization)
*/

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

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
func (s *Store) SaveNodeTexts(uuid string, shard string, texts map[string]string) error {

	for textPath, text := range texts {
		path := s.textPath(shard, uuid, textPath)

		known, err := s.IsFileKnown(path)

		if err != nil {
			return err
		}

		rd := strings.NewReader(text)
		sha1, err2 := s.Transaction.WriteHashObject(rd)
		if err2 != nil {
			return err2
		}

		if known {
			err = s.Transaction.UpdateIndexCache(sha1, path)
		} else {
			err = s.Transaction.AddIndexCache(sha1, path)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) saveBlobToFile(path string, blob io.Reader) error {
	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	// make a buffer to keep chunks that are read
	buf := make([]byte, 1024)
	for {
		// read a chunk
		n, err := blob.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		// write a chunk
		if _, err := file.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}

// map poolname => []nodeUuid, only the blobs that have a key set are going to be changed
func (s *Store) SaveNodeBlobs(uuid string, shard string, blobs map[string]io.Reader) error {

	for blobPath, blob := range blobs {
		path := filepath.Join(s.Git.Dir, s.blobPath(shard, uuid, blobPath))
		if err := s.saveBlobToFile(path, blob); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) callwithBlob(uuid string, shard string, blobPath string, fn func(string, io.Reader) error) error {
	path := filepath.Join(s.Git.Dir, s.blobPath(shard, uuid, blobPath))
	if FileExists(path) {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		return fn(blobPath, file)
	}
	return nil
}

// GetNodeBlobs calls fn for each existing blob in requestedBlobs
func (s *Store) GetNodeBlobs(uuid string, shard string, requestedBlobs []string, fn func(string, io.Reader) error) error {
	for _, blob := range requestedBlobs {
		err := s.callwithBlob(uuid, shard, blob, fn)

		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetNodeTexts(uuid string, shard string, requestedTexts []string) (texts map[string]string, err error) {
	texts = map[string]string{}
	var known bool
	for _, text := range requestedTexts {
		known, err = s.IsFileKnown(text)
		if err != nil {
			return
		}
		if known {
			var buf bytes.Buffer
			err = s.ReadCatHeadFile(s.textPath(shard, uuid, text), &buf)
			if err != nil {
				return
			}
			texts[text] = buf.String()
		}
	}
	return
}

func (s *Store) SaveEdges(category, shard, uuid string, edges map[string]string) error {
	path := s.edgePath(category, shard, uuid)
	known, err := s.IsFileKnown(path)
	if err != nil {
		return err
	}
	return s.save(path, !known, edges)
}

// RemoveEdges also removes the properties node of an edge
// Is the edges file is already removed, no error should be returned
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

// if there is no edge file for the given category, no error is returned, but empty  edges map
func (s *Store) GetEdges(category, shard, uuid string) (edges map[string]string, err error) {
	path := s.edgePath(category, shard, uuid)
	edges = map[string]string{}

	known, err := s.IsFileKnown(path)
	if err != nil {
		return edges, err
	}

	if !known {
		return edges, nil
	}
	err = s.load(path, edges)
	return edges, err
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
// if no node properties file does exist, no error should be returned and s.save(....isNew) should be used
func (g *Store) SaveNodeProperties(uuid string, shard string, props map[string]interface{}) error {
	path := g.propPath(shard, uuid)

	known, err := g.IsFileKnown(path)
	if err != nil {
		return err
	}

	if known {
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

	return g.save(path, !known, props)
}

// TODO Rollback any actions that have been taken since the last commit
// stage should be cleared and any newly added data should be removed
// maybe a cleanup command should remove the orphaned sha1s (git gc maybe??)
func (g *Store) Rollback() error {
	return g.ResetToHeadAll()
}

// TODO what happens on errors? changes will not be committed!
// TODO what about the edges? must delete them all (and identify them all)
// to identify them we need something like
// ls refs/*/shard/uuid[:2], uuid[2:]
// return fmt.Sprintf("refs/%s/%s/%s/%s", category, shard, uuid[:2], uuid[2:])
// if any file does not exist, no error should be returned
func (g *Store) RemoveNode(uuid string, shard string) error {
	// fmt.Printf("trying to remove node: uuid %#v shard %#v\n", uuid, shard)
	// fmt.Println("proppath is ", g.propPath(shard, uuid))
	paths := []string{
		g.propPath(shard, uuid),
		fmt.Sprintf("text/%s/%s/%s", shard, uuid[:2], uuid[2:]),
		fmt.Sprintf("blob/%s/%s/%s", shard, uuid[:2], uuid[2:]),
	}

	files, err := g.LsFiles(fmt.Sprintf("refs/*/%s/%s/%s", shard, uuid[:2], uuid[2:]))
	if err != nil {
		// fmt.Println("error from ls files")
		return err
	}

	for _, file := range files {
		err := g.Transaction.RmIndex(file)
		if err != nil {
			// fmt.Printf("can't remove index: %#v\n", file)
			return err
		}
	}

	for _, path := range paths {
		known, err := g.IsFileKnown(path)
		if err != nil {
			return err
		}
		if known {
			err := g.Transaction.RmIndex(path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// only the properties that exist make it into the returned map
// it is no error if a requested property does not exist for a node
// the caller has to check the returned map against the requested props if
// she wants to check, if all requested properties have been returned
// if the node properties file is not there no error should be returned
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
