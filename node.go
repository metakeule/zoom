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

func (n *Node) LoadBlobs(st Store, requestedBlobs []string, fn func(string, io.Reader) error) (err error) {
	if len(requestedBlobs) > 0 {
		err := st.GetNodeBlobs(n.UUID, n.Shard, requestedBlobs, fn)
		if err != nil {
			return err
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
	} else {
		n.UUID = id[pos+1:]
		n.Shard = id[:pos]
	}
	return n
}

func (n *Node) ID() string {
	return n.Shard + "-" + n.UUID
}

func (n *Node) GetBlob(blob string) io.Reader { return n.blobs[blob] }
func (n *Node) GetText(text string) string    { return n.texts[text] }
func (n *Node) GetBool(prop string) bool      { return n.props[prop].(bool) }
func (n *Node) GetBools(prop string) []bool {
	if n.props[prop] == nil {
		return nil
	}
	return n.props[prop].([]bool)
}
func (n *Node) GetInt(prop string) int64 { return n.props[prop].(int64) }
func (n *Node) GetInts(prop string) []int64 {
	if n.props[prop] == nil {
		return nil
	}
	return n.props[prop].([]int64)
}
func (n *Node) GetFloat(prop string) float64 { return n.props[prop].(float64) }
func (n *Node) GetFloats(prop string) []float64 {
	if n.props[prop] == nil {
		return nil
	}
	return n.props[prop].([]float64)
}
func (n *Node) GetString(prop string) string { return n.props[prop].(string) }
func (n *Node) GetStrings(prop string) []string {
	if n.props[prop] == nil {
		return nil
	}
	return n.props[prop].([]string)
}
func (n *Node) GetTime(prop string) time.Time { return n.props[prop].(time.Time) }
func (n *Node) GetTimes(prop string) []time.Time {
	if n.props[prop] == nil {
		return nil
	}
	return n.props[prop].([]time.Time)
}

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

func (n *Node) SaveTexts(st Store) (err error) {
	saveTexts := map[string]string{}

	for textKey, textVal := range n.texts {
		if n.dirty[textKey] {
			saveTexts[textKey] = textVal
		}
	}

	if len(saveTexts) > 0 {
		err = st.SaveNodeTexts(n.UUID, n.Shard, saveTexts)
		if err != nil {
			return err
		}
	}

	for textKey := range n.texts {
		delete(n.dirty, textKey)
	}

	return nil
}

func (n *Node) SaveBlobs(st Store) (err error) {
	saveBlobs := map[string]io.Reader{}

	for blobKey, blobVal := range n.blobs {
		if n.dirty[blobKey] {
			saveBlobs[blobKey] = blobVal
		}
	}

	if len(saveBlobs) > 0 {
		err = st.SaveNodeBlobs(n.UUID, n.Shard, saveBlobs)
		if err != nil {
			return err
		}
	}

	for blobKey := range n.blobs {
		delete(n.dirty, blobKey)
	}

	return nil
}

func (n *Node) Save(st Store) (err error) {
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
		err = st.SaveNodeProperties(n.UUID, n.Shard, saveProps)
		if err != nil {
			return err
		}
	}

	if len(saveTexts) > 0 {
		err = st.SaveNodeTexts(n.UUID, n.Shard, saveTexts)
		if err != nil {
			return err
		}
	}

	if len(saveBlobs) > 0 {
		err = st.SaveNodeBlobs(n.UUID, n.Shard, saveBlobs)
		if err != nil {
			return err
		}
	}

	n.dirty = map[string]bool{}
	return nil
}

func (n *Node) Remove(st Store) (err error) {
	return st.RemoveNode(n.UUID, n.Shard)
}

// NewEdge creates a new Edge to the target edge, by the way creating a property node based on the given
// properties. The property node is part of the same shard as Node
func (n *Node) NewEdge(st Store, category string, to *Node, props map[string]interface{}) error {
	if len(props) == 0 {
		edge := NewEdge(category, n, to, nil)
		return edge.Save(st)
	}
	propNode := NewNode(n.Shard)
	propNode.props = props

	for k, _ := range props {
		propNode.dirty[k] = true
	}

	if err := propNode.Save(st); err != nil {
		return err
	}

	edge := NewEdge(category, n, to, propNode)
	return edge.Save(st)
}

// RemoveEdge removes the edge of the given category, removing the property node of the edge
func (n *Node) RemoveEdge(st Store, category string, to *Node) error {
	edges, err := st.GetEdges(category, n.Shard, n.UUID)
	if err != nil {
		return err
	}
	if len(edges) == 0 {
		return nil
	}

	propID, has := edges[to.ID()]

	if !has {
		return nil
	}

	propNode := NewNode(propID)
	if err := propNode.Remove(st); err != nil {
		return err
	}
	delete(edges, to.ID())
	if len(edges) == 0 {
		return st.RemoveEdges(category, n.Shard, n.UUID)
	}
	return st.SaveEdges(category, n.Shard, n.UUID, edges)
}

// GetEdge returns nil, if the edge could not be found, does not load the properties of the property edge
func (n *Node) GetEdge(st Store, category string, to *Node) (*Edge, error) {
	edges, err := st.GetEdges(category, n.Shard, n.UUID)
	if err != nil {
		return nil, err
	}
	if len(edges) == 0 {
		return nil, nil
	}

	propID, has := edges[to.ID()]

	if !has {
		return nil, nil
	}

	if propID == "" {
		return NewEdge(category, n, to, nil), nil
	}

	return NewEdge(category, n, to, NewNode(propID)), nil
}

// GetEdges returns all edges for the given category. it does however not load the properties neither
// of the property node nor of the target node
func (n *Node) GetEdges(st Store, category string) ([]*Edge, error) {
	edges, err := st.GetEdges(category, n.Shard, n.UUID)
	if err != nil {
		return nil, err
	}
	if len(edges) == 0 {
		return nil, nil
	}

	res := make([]*Edge, len(edges))

	i := 0
	for toID, propID := range edges {
		res[i] = NewEdge(category, n, NewNode(toID), NewNode(propID))
		i++
	}

	return res, nil
}
