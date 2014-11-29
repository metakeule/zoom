package zoom

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-contrib/uuid"
)

type Node struct {
	UUID  string
	Shard string
	props map[string]interface{} // saved in nodes file
	texts map[string]string      // saved in each file for a text (text is string lenghth > 255) texts are always UTF-8, \n
	blobs map[string]io.Reader   // saved outside the repo inside the working dir (will be synced via rsync), blobpath must begin with mimetype
	dirty map[string]bool
	IsNew bool
}

func (n *Node) LoadProperties(st Store, requestedProps []string) (err error) {
	if len(requestedProps) > 0 {
		props, err := st.GetNodeProperties(n.UUID, n.Shard, requestedProps)

		if err != nil {
			return err
		}

		for k, v := range props {
			n.props[k] = v
			n.dirty[k] = false
		}
	}
	return nil
}

func (n *Node) LoadTexts(st Store, requestedTexts []string) (err error) {
	if len(requestedTexts) > 0 {
		texts, err := st.GetNodeTexts(n.UUID, n.Shard, requestedTexts)

		if err != nil {
			return err
		}

		for k, v := range texts {
			n.texts[k] = v
			n.dirty[k] = false
		}
	}
	return nil
}

func (n *Node) LoadBlobs(st Store, requestedBlobs []string) (err error) {
	if len(requestedBlobs) > 0 {
		blobs, err := st.GetNodeBlobs(n.UUID, n.Shard, requestedBlobs)

		if err != nil {
			return err
		}

		for k, v := range blobs {
			n.blobs[k] = v
			n.dirty[k] = false
		}
	}
	return nil
}

func SplitID(id string) (shard, uuid string, err error) {
	pos := strings.Index(id, "-")
	if pos == -1 {
		err = fmt.Errorf("invalid id %#v", id)
	} else {
		uuid = id[pos+1:]
		shard = id[:pos]
	}
	return
}

// NewNode creates a new node
// id is a string consisting of the shardname (may only contain the characters
// ([a-z][a-z0-9]+), a - sign and a uuid
// if the given id has no - sign, it is considered that the id is the shard and
// the uuid has to be generated.
// otherwise shard and uuid will be split off the id
// the id returned by ID() can be passed to NewNode() in order to load a node
func NewNode(id string) *Node {
	n := &Node{
		dirty: map[string]bool{},
		props: map[string]interface{}{},
		texts: map[string]string{},
		blobs: map[string]io.Reader{},
	}
	pos := strings.Index(id, "-")
	if pos == -1 {
		n.Shard = id
		n.UUID = uuid.NewV4().String()
		n.IsNew = true
	} else {
		n.UUID = id[pos+1:]
		n.Shard = id[:pos]
	}
	return n
}

func (n *Node) ID() string {
	return n.Shard + "-" + n.UUID
}

func (n *Node) GetBlob(blob string) io.Reader    { return n.blobs[blob] }
func (n *Node) GetText(text string) string       { return n.texts[text] }
func (n *Node) GetBool(prop string) bool         { return n.props[prop].(bool) }
func (n *Node) GetBools(prop string) []bool      { return n.props[prop].([]bool) }
func (n *Node) GetInt(prop string) int64         { return n.props[prop].(int64) }
func (n *Node) GetInts(prop string) []int64      { return n.props[prop].([]int64) }
func (n *Node) GetFloat(prop string) float64     { return n.props[prop].(float64) }
func (n *Node) GetFloats(prop string) []float64  { return n.props[prop].([]float64) }
func (n *Node) GetString(prop string) string     { return n.props[prop].(string) }
func (n *Node) GetStrings(prop string) []string  { return n.props[prop].([]string) }
func (n *Node) GetTime(prop string) time.Time    { return n.props[prop].(time.Time) }
func (n *Node) GetTimes(prop string) []time.Time { return n.props[prop].([]time.Time) }

// SetBlob stores a binary large object
func (o *Node) SetBlob(prop string, rc io.Reader) {
	o.dirty[prop] = true
	o.blobs[prop] = rc
}

// SetText stores larger strings that can be more than 255 bytes long
func (o *Node) SetText(prop string, val string) {
	o.dirty[prop] = true
	o.texts[prop] = val
}

func (o *Node) SetBool(prop string, val bool) {
	o.dirty[prop] = true
	o.props[prop] = val
}

func (o *Node) SetBools(prop string, vals ...bool) {
	o.dirty[prop] = true
	o.props[prop] = vals
}

func (o *Node) SetInt(prop string, val int64) {
	o.dirty[prop] = true
	o.props[prop] = val
}

func (o *Node) SetInts(prop string, vals ...int64) {
	o.dirty[prop] = true
	o.props[prop] = vals
}

func (o *Node) SetFloat(prop string, val float64) {
	o.dirty[prop] = true
	o.props[prop] = val
}

func (o *Node) SetFloats(prop string, vals ...float64) {
	o.dirty[prop] = true
	o.props[prop] = vals
}

// SetString sets a string that has the max length of 255 bytes.
// a larger string returns an error
func (o *Node) SetString(prop string, val string) error {
	if len(val) > 255 {
		return fmt.Errorf("string %#v is too large for SetString value, use SetText", val)
	}
	o.dirty[prop] = true
	o.props[prop] = val
	return nil
}

// SetStrings sets strings that have the max length of 255 bytes.
// larger strings return an error
func (o *Node) SetStrings(prop string, vals ...string) error {

	for _, s := range vals {
		if len(s) > 255 {
			return fmt.Errorf("string %#v is too large for SetString value, use SetText", s)
		}

	}

	o.dirty[prop] = true
	o.props[prop] = vals
	return nil
}

func (o *Node) SetTime(prop string, val time.Time) {
	o.dirty[prop] = true
	o.props[prop] = val
}

func (o *Node) SetTimes(prop string, vals ...time.Time) {
	o.dirty[prop] = true
	o.props[prop] = vals
}

// TODO: offer a way to just save Properties, Relations or Pools
func (n *Node) Save(st Store) (err error) {
	// dirty, props, rels, pools := n.Dirty(), n.Properties(), n.Relations(), n.Pools()
	saveProps, saveTexts, saveBlobs := map[string]interface{}{}, map[string]string{}, map[string]io.Reader{}

	for key, isDirty := range n.dirty {
		if isDirty {
			prop, isProp := n.props[key]
			if isProp {
				saveProps[key] = prop
				continue
			}

			text, isText := n.texts[key]
			if isText {
				saveTexts[key] = text
				continue
			}

			blob, isBlob := n.blobs[key]
			if isBlob {
				saveBlobs[key] = blob
				continue
			}
		}
	}

	if len(saveProps) > 0 {
		err = st.SaveNodeProperties(n.UUID, n.Shard, n.IsNew, saveProps)
		if err != nil {
			return err
		}
	}

	if len(saveTexts) > 0 {
		err = st.SaveNodeTexts(n.UUID, n.Shard, n.IsNew, saveTexts)
		if err != nil {
			return err
		}
	}

	if len(saveBlobs) > 0 {
		err = st.SaveNodeBlobs(n.UUID, n.Shard, n.IsNew, saveBlobs)
		if err != nil {
			return err
		}
	}

	n.dirty = map[string]bool{}
	n.IsNew = false
	return nil
}

func (n *Node) Remove(st Store) (err error) {
	return st.RemoveNode(n.UUID, n.Shard)
}
