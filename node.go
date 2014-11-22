package zoom

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-contrib/uuid"
)

type Pool []*Node

type Node struct {
	uuid string
	// Data may be Properties and relations to other *nodes or Pools
	data  map[string]interface{}
	dirty map[string]bool
	isNew bool
	shard string
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
		data:  map[string]interface{}{},
	}
	pos := strings.Index(id, "-")
	if pos == -1 {
		n.shard = id
		n.uuid = uuid.NewV4().String()
		n.isNew = true
	} else {
		n.uuid = id[pos+1:]
		n.shard = id[:pos]
	}
	return n
}

func (n *Node) ID() string {
	return n.shard + "-" + n.uuid
}

func (n *Node) Shard() string {
	return n.shard
}

func (o *Node) Set(s map[string]interface{}) {
	for k, _ := range s {
		o.dirty[k] = true
	}
	o.data = s
}

func (o *Node) Update(key string, val interface{}) {
	o.dirty[key] = true
	o.data[key] = val
}

func (n *Node) Dirty() map[string]bool {
	return n.dirty
}

func (n *Node) ClearDirty() {
	n.dirty = map[string]bool{}
}

func (o *Node) Property(p Property) (has bool) {
	return p.Get(o.data)
}

func (o *Node) Properties() map[string]interface{} {
	props := map[string]interface{}{}

	for name, d := range o.data {
		switch ty := d.(type) {
		case *Node, Node:
			continue
		case nil:
			continue
		case int, int64, float64, time.Time, string, []int, []float64, []time.Time, []string:
			props[name] = ty
		default:
			panic(fmt.Sprintf("type %T not allowed in Data", d))
		}
	}

	return props
}

func (o *Node) Relations() map[string]*Node {
	rels := map[string]*Node{}
	for name, d := range o.data {
		if rel, ok := d.(*Node); ok {
			rels[name] = rel
		}
	}
	return rels
}

func (o *Node) Pools() map[string]Pool {
	rels := map[string]Pool{}
	for name, d := range o.data {
		if rel, ok := d.(Pool); ok {
			rels[name] = rel
		}
	}
	return rels
}

func (o *Node) Pool(k string) Pool {
	oo, has := o.data[k]
	if !has {
		return nil
	}
	return oo.(Pool)
}

func (o *Node) Relation(k string) *Node {
	oo, has := o.data[k]
	if !has {
		return nil
	}
	return oo.(*Node)
}

func (n *Node) LoadProperties(st Store, requestedProps []string) (err error) {
	if len(requestedProps) > 0 {
		props, err := st.GetNodeProperties(n.ID(), requestedProps)

		if err != nil {
			return err
		}

		for k, v := range props {
			n.data[k] = v
		}
	}
	return nil
}

func (n *Node) LoadRelations(st Store, relations map[string][]string) (err error) {
	if len(relations) > 0 {

		requestedRels := make([]string, len(relations))

		i := 0
		for k, _ := range relations {
			requestedRels[i] = k
			i++
		}

		rels, err := st.GetNodeRelations(n.ID(), requestedRels)

		if err != nil {
			return err
		}

		for k, fields := range relations {
			id := rels[k]
			props, err := st.GetNodeProperties(id, fields)
			if err != nil {
				return err
			}

			relNode := NewNode(id)
			relNode.Set(props)
			n.data[k] = relNode
		}

	}
	return nil
}

func (n *Node) LoadPools(st Store, pools map[string][]string) (err error) {
	if len(pools) > 0 {

		requestedPools := make([]string, len(pools))

		i := 0
		for k, _ := range pools {
			requestedPools[i] = k
			i++
		}

		pls, err := st.GetNodePools(n.ID(), requestedPools)

		if err != nil {
			return err
		}

		for k, fields := range pools {
			ids := pls[k]

			nodes := make([]*Node, len(ids))

			for i, id := range ids {

				props, err := st.GetNodeProperties(id, fields)
				if err != nil {
					return err
				}

				poolNode := NewNode(id)
				poolNode.Set(props)
				nodes[i] = poolNode
			}

			n.data[k] = Pool(nodes)
		}

	}
	return nil
}

// TODO: offer a way to just save Properties, Relations or Pools
func (n *Node) Save(st Store) (err error) {
	dirty, props, rels, pools := n.Dirty(), n.Properties(), n.Relations(), n.Pools()
	saveProps, saveRels, savePools := map[string]interface{}{}, map[string]string{}, map[string][]string{}

	for key, isDirty := range dirty {
		if isDirty {
			prop, isProp := props[key]
			if isProp {
				saveProps[key] = prop
				continue
			}

			rel, isRel := rels[key]
			if isRel {
				saveRels[key] = rel.ID()
				continue
			}

			pool, isPool := pools[key]
			if isPool {
				poolIDs := make([]string, len(pool))
				for i, nd := range pool {
					poolIDs[i] = nd.ID()
				}
				savePools[key] = poolIDs
				continue
			}
		}
	}

	// if len(saveProps) > 0 {
	err = st.SaveNodeProperties(n.ID(), n.isNew, saveProps)
	if err != nil {
		return err
	}
	// }

	// if len(saveRels) > 0 {
	err = st.SaveNodeRelations(n.ID(), n.isNew, saveRels)
	if err != nil {
		return err
	}
	// }

	// if len(savePools) > 0 {
	err = st.SaveNodePools(n.ID(), n.isNew, savePools)
	if err != nil {
		return err
	}
	// }

	n.ClearDirty()
	n.isNew = false
	return nil
}

func (n *Node) Remove(st Store) (err error) {
	return st.RemoveNode(n.ID())
}
