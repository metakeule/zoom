package zoom

type Property interface {
	Get(data map[string]interface{}) (has bool)
}

type Name string

func (n *Name) Get(data map[string]interface{}) bool {
	nn, has := data["name"]
	if !has {
		return false
	}
	nstr, ok := nn.(string)
	if !ok {
		return false
	}
	*n = Name(nstr)
	return true
}

// TODO make shortcuts for common things like a Nameable struct, sticky things, children with parents, relations with properties
// TODO calculate distances from a node to another via relations that have the requested key
// TODO make shortcuts to add a new node. remove a node and update a node (with validation and store)
// TODO make indexed collections to find nodes based on certain criterions, make rules to insert, update, remove
// entries if they are indexed
// TODO make LifeSpan structs that have a BirthDate and a DeadDate (then nothing will change)
// TODO make Log structs that have an author and a DateOfChange and that log every interesting change
// that was made by author and date
// TODO make git a storage option
// TODO make a Group as struct that is a relation of nodes belonging together and forming a whole (e.g. collection of properties
// that belong to different schemata)
// TODO make a delegator that delegates ("inherits") properties from another node
// create an interface in places where *O is used

type Identifiable interface {
	ID() string
}

type Nameable interface {
	Name() string
}

type StructerFunc func(*Node) (stru Identifiable, ok bool)

func (s StructerFunc) Struct(o *Node) (stru Identifiable, ok bool) {
	return s(o)
}

//type MkStruct func(*O) (stru interface{}, ok bool)
type Structer interface {
	Struct(*Node) (stru Identifiable, ok bool)
}

// StructerFallback tries each Structer until one matches
type StructerFallback []Structer

func (f StructerFallback) Struct(o *Node) (stru Identifiable, ok bool) {
	for _, str := range f {
		stru, ok = str.Struct(o)
		if ok {
			return
		}
	}
	return
}

type Handler interface {
	Handle(stru Identifiable) (ok bool)
}

// HandlerFallback tries each Handler until one did handle
type HandlerFallback []Handler

func (h HandlerFallback) Handle(stru Identifiable) (ok bool) {
	for _, hdl := range h {
		if hdl.Handle(stru) {
			return true
		}
	}

	return false
}

// DistinctMix has Os that can be just one Struct type
// only Os that can be transformed to a struct will be handled
type DistinctMix struct {
	Nodes []*Node
	StructerFallback
	HandlerFallback
}

// Handle handles each O that can be handled and returns the handled *Os
// and the unhandled *Os
func (d DistinctMix) Handle() (handled, unhandled []*Node) {
	for _, oo := range d.Nodes {
		stru, ok := d.StructerFallback.Struct(oo)
		if ok {
			if d.HandlerFallback.Handle(stru) {
				handled = append(handled, oo)
				continue
			}
		}
		unhandled = append(unhandled, oo)
	}
	return
}

/*
type Structer interface {
	Name() string
	MkStruct(*O) (stru interface{}, ok bool)
}
'/'

//func(*O) (stru interface{}, ok bool)

var structerfuncs []StructerFunc

func Register(fn StructerFunc) {
	structerfuncs = append(structerfuncs, fn)
}
*/

// func MkStruct(*O, func(*O) (stru interface{}, ok bool)) interface{} {
//
// }
