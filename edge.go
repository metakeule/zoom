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
