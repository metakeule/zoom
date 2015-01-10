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

func (e *Edge) Save() error {
	edges, err := e.From.Transaction.GetEdges(e.Category, e.From.Id)
	if err != nil {
		return err
	}
	if e.Properties != nil {
		edges[e.To.Transaction.Shard()+"-"+e.To.Id] = e.Properties.Id
	} else {
		edges[e.To.Transaction.Shard()+"-"+e.To.Id] = ""
	}
	return e.From.Transaction.SaveEdges(e.Category, e.From.Id, edges)
}

// Remove only removes the Edge entry inside the from node edges, but not the property node of the edges
func (e *Edge) Remove() error {
	edges, err := e.From.Transaction.GetEdges(e.Category, e.From.Id)
	if err != nil {
		return err
	}
	delete(edges, e.To.Transaction.Shard()+"-"+e.To.Id)
	if len(edges) == 0 {
		return e.From.Transaction.RemoveEdges(e.Category, e.From.Id)
	}
	return e.From.Transaction.SaveEdges(e.Category, e.From.Id, edges)
}
