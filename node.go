package zoom

import (
	"fmt"
	"time"

	"github.com/go-contrib/uuid"
)

type Node interface {
	UUID() string
	Update(string, interface{})
	Set(map[string]interface{})
	SetUUID(uuid string)
	Property(p Property) (has bool)
	Properties() map[string]interface{}
	Pools() map[string]Pool
	Pool(k string) Pool
	// return the dirty keys
	Dirty() map[string]bool
	// clears the dirty keys
	ClearDirty()
	Relations() map[string]Node
	Relation(k string) Node
	Save(Store) error
	Remove(Store) error
	LoadProperties(st Store, requestedProps []string) (err error)
	LoadRelations(st Store, relations map[string][]string) (err error)
	LoadPools(st Store, pools map[string][]string) (err error)
}

var _ Node = &node{}

type Pool []Node

type node struct {
	uuid string
	// Data may be Properties and relations to other *nodes or Pools
	data map[string]interface{}

	dirty map[string]bool

	isNew bool
}

func NewNode() Node {
	return &node{
		dirty: map[string]bool{},
		data:  map[string]interface{}{},
		uuid:  uuid.NewV4().String(),
		isNew: true,
	}
}

func (o *node) UUID() string {
	return o.uuid
}

func (o *node) SetUUID(uuid string) {
	if o == nil {
		o = &node{
			dirty: map[string]bool{},
			data:  map[string]interface{}{},
		}
	}
	o.uuid = uuid
}

func (o *node) Set(s map[string]interface{}) {
	for k, _ := range s {
		o.dirty[k] = true
	}
	o.data = s
}

func (o *node) Update(key string, val interface{}) {
	o.dirty[key] = true
	o.data[key] = val
}

func (n *node) Dirty() map[string]bool {
	return n.dirty
}

func (n *node) ClearDirty() {
	n.dirty = map[string]bool{}
}

func (o *node) Property(p Property) (has bool) {
	return p.Get(o.data)
}

func (o *node) Properties() map[string]interface{} {
	props := map[string]interface{}{}

	for name, d := range o.data {
		switch ty := d.(type) {
		case *node, Node:
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

func (o *node) Relations() map[string]Node {
	rels := map[string]Node{}
	for name, d := range o.data {
		if rel, ok := d.(*node); ok {
			rels[name] = rel
		}
	}
	return rels
}

func (o *node) Pools() map[string]Pool {
	rels := map[string]Pool{}
	for name, d := range o.data {
		if rel, ok := d.(Pool); ok {
			rels[name] = rel
		}
	}
	return rels
}

func (o *node) Pool(k string) Pool {
	oo, has := o.data[k]
	if !has {
		return nil
	}
	return oo.(Pool)
}

func (o *node) Relation(k string) Node {
	oo, has := o.data[k]
	if !has {
		return nil
	}
	return oo.(*node)
}

func (n *node) LoadProperties(st Store, requestedProps []string) (err error) {
	if len(requestedProps) > 0 {
		props, err := st.GetNodeProperties(n.UUID(), requestedProps)

		if err != nil {
			return err
		}

		for k, v := range props {
			n.data[k] = v
		}
	}
	return nil
}

func (n *node) LoadRelations(st Store, relations map[string][]string) (err error) {
	if len(relations) > 0 {

		requestedRels := make([]string, len(relations))

		i := 0
		for k, _ := range relations {
			requestedRels[i] = k
			i++
		}

		rels, err := st.GetNodeRelations(n.UUID(), requestedRels)

		if err != nil {
			return err
		}

		for k, fields := range relations {
			uuid := rels[k]
			props, err := st.GetNodeProperties(uuid, fields)
			if err != nil {
				return err
			}

			relNode := NewNode()
			relNode.SetUUID(uuid)
			relNode.Set(props)
			n.data[k] = relNode
		}

	}
	return nil
}

func (n *node) LoadPools(st Store, pools map[string][]string) (err error) {
	if len(pools) > 0 {

		requestedPools := make([]string, len(pools))

		i := 0
		for k, _ := range pools {
			requestedPools[i] = k
			i++
		}

		pls, err := st.GetNodePools(n.UUID(), requestedPools)

		if err != nil {
			return err
		}

		for k, fields := range pools {
			uuids := pls[k]

			nodes := make([]Node, len(uuids))

			for i, uuid := range uuids {

				props, err := st.GetNodeProperties(uuid, fields)
				if err != nil {
					return err
				}

				poolNode := NewNode()
				poolNode.SetUUID(uuid)
				poolNode.Set(props)
				nodes[i] = poolNode
			}

			n.data[k] = Pool(nodes)
		}

	}
	return nil
}

// TODO: offer a way to just save Properties, Relations or Pools
func (n *node) Save(st Store) (err error) {
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
				saveRels[key] = rel.UUID()
				continue
			}

			pool, isPool := pools[key]
			if isPool {
				poolUUIDs := make([]string, len(pool))
				for i, nd := range pool {
					poolUUIDs[i] = nd.UUID()
				}
				savePools[key] = poolUUIDs
				continue
			}
		}
	}

	// if len(saveProps) > 0 {
	err = st.SaveNodeProperties(n.UUID(), n.isNew, saveProps)
	if err != nil {
		return err
	}
	// }

	// if len(saveRels) > 0 {
	err = st.SaveNodeRelations(n.UUID(), n.isNew, saveRels)
	if err != nil {
		return err
	}
	// }

	// if len(savePools) > 0 {
	err = st.SaveNodePools(n.UUID(), n.isNew, savePools)
	if err != nil {
		return err
	}
	// }

	n.ClearDirty()
	n.isNew = false
	return nil
}

func (n *node) Remove(st Store) (err error) {
	return st.RemoveNode(n.UUID())
}
