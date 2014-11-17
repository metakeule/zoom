package zoom

type Store interface {

	// rollback the actions since the last successfull commit
	Rollback() error

	// only the props that have a key set are going to be changed
	SaveNodeProperties(nodeUuid string, isNew bool, props map[string]interface{}) error

	// map relname => nodeUuid, only the relations that have a key set are going to be changed
	SaveNodeRelations(nodeUuid string, isNew bool, relations map[string]string) error

	// map poolname => []nodeUuid, only the pools that have a key set are going to be changed
	SaveNodePools(nodeUuid string, isNew bool, pools map[string][]string) error

	// remove node with properties, relations and pools
	// references are not checked nor deleted, cascading deletes must be made from the outside
	RemoveNode(nodeUuid string) error

	// get the node from the store. the given node contains all properties
	// that are searched/known, e.g. the uuid
	// all properties will be set inside the node and also the uuid it is was not already set
	//GetNode(queryAndResult Node) error

	// maybe just make queries part of an index and not part of the store

	// only the properties that exist make it into the returned map
	// it is no error if a requested property does not exist for a node
	// the caller has to check the returned map against the requested props if
	// she wants to check, if all requested properties have been returned
	GetNodeProperties(nodeUuid string, requestedProps []string) (props map[string]interface{}, err error)

	// the returned map has as values the uuids of the nodes
	// only the relations that exist make it into the returned map
	// it is no error if a requested relation does not exist for a node
	// the caller has to check the returned map against the requested rels if
	// she wants to check, if all requested relations have been returned
	// also there is no guarantee that the nodes which uuids are returned do still exist.
	// there must be wrappers put around the store to ensure this (preferably by using indices)
	GetNodeRelations(nodeUuid string, requestedRels []string) (rels map[string]string, err error)

	// the returned map has as values slices of uuids of the nodes
	// only the pools that exist make it into the returned map
	// it is no error if a requested pool does not exist for a node
	// the caller has to check the returned map against the requested pools if
	// she wants to check, if all requested pools have been returned
	// also there is no guarantee that the nodes which uuids are returned do still exist.
	// there must be wrappers put around the store to ensure this (preferably by using indices)
	GetNodePools(nodeUuid string, requestedPools []string) (pools map[string][]string, err error)

	// save the changes in the db
	Commit(comment string) error

	// GetNodes returns all nodes that conform to the given query, where
	// each query node is combined by OR whereas each Nodes properties and relations are combined
	// by and AND
	// properties and relations set to nil are considered as: must have property/relation, but with no defined
	// value
	// more complex query must be constructed via special indices that are queried
	// maybe just make queries part of an index and not part of the store
	// GetNodes(query ...Node) ([]Node, error)

	/*
		// GetPool returns a pool based on its uuid
		GetPool(uuid string) (*Pool, error)

		// SavePool saves the given pool based on its uuid
		SavePool(*Pool) error

		// removes the given node and returns any error
		RemovePool(Node) error
	*/

}
