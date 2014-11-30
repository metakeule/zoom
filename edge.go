package zoom

type Edge struct {
	Category   string
	From       *Node
	To         *Node
	Properties *Node
}

func NewEdge(category string, from, to, properties *Node) *Edge {
	return &Edge{
		Category:   category,
		From:       from,
		To:         to,
		Properties: properties,
	}
}

func (e *Edge) Save(st Store) error {
	edges, err := st.GetEdges(e.Category, e.From.Shard, e.From.UUID)
	if err != nil {
		return err
	}
	edges[e.To.ID()] = e.Properties.ID()
	return st.SaveEdges(e.Category, e.From.Shard, e.From.UUID, edges)
}

// Remove only removes the Edge entry inside the from node edges, but not the property node of the edges
func (e *Edge) Remove(st Store) error {
	edges, err := st.GetEdges(e.Category, e.From.Shard, e.From.UUID)
	if err != nil {
		return err
	}
	delete(edges, e.To.ID())
	if len(edges) == 0 {
		return st.RemoveEdges(e.Category, e.From.Shard, e.From.UUID)
	}
	return st.SaveEdges(e.Category, e.From.Shard, e.From.UUID, edges)
}
