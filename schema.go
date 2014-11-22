package zoom

type SchemaRules []string

func (s *SchemaRules) Get(data map[string]interface{}) bool {
	nn, has := data["schema-rules"]
	if !has {
		return false
	}
	nstr, ok := nn.([]string)
	if !ok {
		return false
	}
	*s = nstr
	return true
}

var schemaRules = map[string]func(o *Node) error{}

func RegisterSchemaRule(name string, fn func(o *Node) error) {
	schemaRules[name] = fn
}

type Schema struct {
	id    string
	name  string
	rules []string // rule names as registered via RegisterSchemaRule
}

func (sc *Schema) ID() string {
	return sc.id
}

func (sc *Schema) Name() string {
	return sc.name
}

func (sc *Schema) Validate(o *Node) error {
	for _, rl := range sc.rules {
		if err := schemaRules[rl](o); err != nil {
			return err
		}
	}
	return nil
}

func MkSchema(o *Node) (stru Identifiable, ok bool) {
	var n Name
	if !o.Property(&n) {
		return nil, false
	}
	var r SchemaRules
	if !o.Property(&r) {
		return nil, false
	}
	return &Schema{
		id:    o.ID(),
		name:  string(n),
		rules: []string(r),
	}, true
}

type Validateable struct {
	id string
	*Node
	*Schema
}

func (v *Validateable) ID() string {
	return v.id
}

func (v *Validateable) Validate() error {
	return v.Schema.Validate(v.Node)
}

func MkValidateable(o *Node) (stru Identifiable, ok bool) {
	rel := o.Relation("schema")
	if rel == nil {
		return nil, false
	}
	sc, is := MkSchema(rel)
	if !is {
		return nil, false
	}
	scc, isOk := sc.(*Schema)
	if !isOk {
		return nil, false
	}

	return &Validateable{
		id:     o.ID(),
		Node:   o,
		Schema: scc,
	}, true

}

var _ StructerFunc = MkSchema
var _ StructerFunc = MkValidateable
