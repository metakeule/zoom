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

TODO check transactions  with indices!!!
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
	shard string
}

func Open(baseDir string, shard string) (g Git, err error) {
	// fmt.Println("opening")

	gitBase := filepath.Join(baseDir, ".git")

	// ignoring error because gitBase might already exist
	// println("creating " + gitBase)
	os.Mkdir(gitBase, 0755)

	var git *gitlib.Git
	git, err = gitlib.NewGit(gitBase)

	if err != nil {
		return
	}
	if !git.IsInitialized() {
		// fmt.Println("initializing")
		err = git.Transaction(func(tx *gitlib.Transaction) error {
			if err := tx.InitBare(); err != nil {
				return err
			}
			/*
				sha1, err := tx.WriteHashObject(strings.NewReader("index\nblob\n"))
				if err != nil {
					return err
				}

				err = tx.AddIndexCache(sha1, ".gitignore")
				if err != nil {
					return err
				}
			*/
			sha1, err := tx.WriteHashObject(strings.NewReader("ZOOM DATABASE\nThis is a zoom database.\nDon't write into this directory manually.\nUse the zoom database library instead.\n"))
			if err != nil {
				return err
			}

			err = tx.AddIndexCache(sha1, "README")
			if err != nil {
				return err
			}

			sha1, err = tx.WriteTree()
			if err != nil {
				return err
			}

			sha1, err = tx.CommitTree(sha1, "", strings.NewReader("add README"))
			if err != nil {
				return err
			}

			return tx.UpdateHeadsRef("master", sha1)

		})
		if err != nil {
			return
		}
	}
	g = Git{Git: git, shard: shard}
	return
}

func (g *Git) Transaction(msg zoom.CommitMessage, action func(zoom.Transaction) error) (err error) {
	return g.Git.Transaction(func(tx *gitlib.Transaction) error {
		var store zoom.Store = &Store{tx, g.shard}
		return zoom.NewTransaction(store, msg, action)
	})
}

type Store struct {
	*gitlib.Transaction
	shard string
}

// map relname => nodeUuid, only the texts that have a key set are going to be changed
func (s *Store) SaveNodeTexts(uuid string, texts map[string]string) error {

	for textPath, text := range texts {
		path := s.textPath(uuid, textPath)

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
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
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

/*
// map poolname => []nodeUuid, only the blobs that have a key set are going to be changed
func (s *Store) SaveNodeBlobs(uuid string, blobs map[string]io.Reader) error {

	for blobPath, blob := range blobs {
		path := filepath.Join(s.Git.Dir, s.BlobPath(uuid, blobPath))
		if err := s.saveBlobToFile(path, blob); err != nil {
			return err
		}
	}
	return nil
}
*/

/*
func (s *Store) SaveIndex(indexpath string, shard string, rd io.Reader) error {
	path := filepath.Join(s.Git.Dir, s.indexPath(shard, indexpath))
	return s.saveBlobToFile(path, rd)
}
*/

/*

*/

/*
func (s *Store) callwithBlob(uuid string, blobPath string, fn func(string, io.Reader) error) error {
	path := filepath.Join(s.Git.Dir, s.blobPath(uuid, blobPath))
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
*/

/*
func (s *Store) GetIndex(indexpath string, shard string, fn func(io.Reader) error) error {
	path := filepath.Join(s.Git.Dir, s.indexPath(shard, indexpath))
	if FileExists(path) {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		return fn(file)
	}
	return nil
}
*/

// GetNodeBlobs calls fn for each existing blob in requestedBlobs
/*
func (s *Store) GetNodeBlobs(uuid string, requestedBlobs []string, fn func(string, io.Reader) error) error {
	for _, blob := range requestedBlobs {
		err := s.callwithBlob(uuid, blob, fn)

		if err != nil {
			return err
		}
	}
	return nil
}
*/

func (s *Store) GetNodeTexts(uuid string, requestedTexts []string) (texts map[string]string, err error) {
	texts = map[string]string{}
	var known bool
	for _, text := range requestedTexts {
		known, err = s.IsFileKnown(s.textPath(uuid, text))
		if err != nil {
			return
		}

		// fmt.Printf("file %s is known: %v\n", text, known)
		if known {
			var buf bytes.Buffer
			err = s.ReadCatHeadFile(s.textPath(uuid, text), &buf)
			if err != nil {
				return
			}
			texts[text] = buf.String()
		}
	}
	return
}

func (s *Store) SaveEdges(category, uuid string, edges map[string]string) error {
	path := s.edgePath(category, uuid)
	known, err := s.IsFileKnown(path)
	if err != nil {
		return err
	}
	return s.save(path, !known, edges)
}

// RemoveEdges also removes the properties node of an edge
// Is the edges file is already removed, no error should be returned
func (s *Store) RemoveEdges(category, uuid string) error {
	edges, err := s.GetEdges(category, uuid)
	if err != nil {
		return err
	}

	for _, propID := range edges {
		if err := s.RemoveNode(propID); err != nil {
			return err
		}
	}

	path := s.edgePath(category, uuid)
	return s.RmIndex(path)
}

// if there is no edge file for the given category, no error is returned, but empty  edges map
func (s *Store) GetEdges(category, uuid string) (edges map[string]string, err error) {
	path := s.edgePath(category, uuid)
	edges = map[string]string{}

	known, err := s.IsFileKnown(path)
	if err != nil {
		return edges, err
	}

	if !known {
		return edges, nil
	}
	err = s.load(path, &edges)
	return edges, err
}

func (s *Store) edgePath(category string, uuid string) string {
	//return fmt.Sprintf("node/props/%s/%s", uuid[:2], uuid[2:])
	return fmt.Sprintf("refs/%s/%s/%s/%s", category, s.shard, uuid[:2], uuid[2:])
}

func (s *Store) propPath(uuid string) string {
	//return fmt.Sprintf("node/props/%s/%s", uuid[:2], uuid[2:])
	return fmt.Sprintf("node/%s/%s/%s", s.shard, uuid[:2], uuid[2:])
}

func (s *Store) textPath(uuid string, key string) string {
	// return fmt.Sprintf("node/rels/%s/%s", uuid[:2], uuid[2:])
	return fmt.Sprintf("text/%s/%s/%s/%s", s.shard, uuid[:2], uuid[2:], key)
}

func (s *Store) BlobPath(uuid string, blobpath string) string {
	return fmt.Sprintf("../blob/%s/%s/%s/%s", s.shard, uuid[:2], uuid[2:], blobpath)
}

func (g *Store) Commit(msg zoom.CommitMessage) error {
	// fmt.Println("commit from store " + comment)
	treeSha, err := g.Transaction.WriteTree()
	if err != nil {
		return err
	}

	var parent string
	parent, err = g.ShowHeadsRef("master")
	// fmt.Println("parent commit is: " + parent)
	if err != nil {
		return err
	}

	var commitSha string
	commitSha, err = g.CommitTree(treeSha, parent, strings.NewReader(msg.String()))

	if err != nil {
		return err
	}

	return g.UpdateHeadsRef("master", commitSha)
}

func (g *Store) save(path string, isNew bool, data interface{}) error {
	// fmt.Printf("storing: %#v in %#v\n", data, path)
	var buf bytes.Buffer
	// enc := msgpack.NewEncoder(&buf)
	enc := json.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return err
	}

	// fmt.Println("result", buf.String())
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
	// fmt.Println("loading from ", path)
	var buf bytes.Buffer
	err := g.Transaction.ReadCatHeadFile(path, &buf)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// fmt.Println("reading", buf.String())
	//dec := msgpack.NewDecoder(&buf)
	dec := json.NewDecoder(&buf)
	return dec.Decode(data)
}

// only the props that have a key set are going to be changed
// if no node properties file does exist, no error should be returned and s.save(....isNew) should be used
func (g *Store) SaveNodeProperties(uuid string, props map[string]interface{}) error {
	path := g.propPath(uuid)

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
func (g *Store) RemoveNode(uuid string) error {
	// fmt.Printf("trying to remove node: uuid %#v shard %#v\n", uuid, shard)
	// fmt.Println("proppath is ", g.propPath(shard, uuid))
	paths := []string{
		g.propPath(uuid),
		fmt.Sprintf("text/%s/%s/%s", g.shard, uuid[:2], uuid[2:]),
		fmt.Sprintf("blob/%s/%s/%s", g.shard, uuid[:2], uuid[2:]),
	}

	files, err := g.LsFiles(fmt.Sprintf("refs/*/%s/%s/%s", g.shard, uuid[:2], uuid[2:]))
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
func (g *Store) GetNodeProperties(uuid string, requestedProps []string) (props map[string]interface{}, err error) {
	path := g.propPath(uuid)
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

func (g *Store) Shard() string {
	return g.shard
}
