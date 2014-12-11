package zoom

import "io"

/*
	props map[string]interface{}   // saved in nodes file
	texts map[string]string        // saved in each file for a text (text is string lenghth > 255) texts are always UTF-8, \n
	blobs map[string]io.Reader // saved outside the repo inside the working dir (will be synced via rsync), blobpath must begin with mimetype
*/

/*
type Shard struct {
	store Store
	name  string
}

func NewShard(name string, st Store) *Shard {
	return &Shard{name: name, store: st}
}

type Transaction struct {
	*Shard
}

func (s *Transaction) SaveNodeProperties(uuid string, props map[string]interface{}) error {
	return s.store.SaveNodeProperties(uuid, s.name, props)
}

func (s *Transaction) SaveNodeTexts(uuid string, texts map[string]string) error {
	return s.store.SaveNodeTexts(uuid, s.name, texts)
}

func (s *Transaction) SaveNodeBlobs(uuid string, blobs map[string]io.Reader) error {
	return s.store.SaveNodeBlobs(uuid, s.name, blobs)
}

func (s *Transaction) SaveEdges(category, fromUUID string, edges map[string]string) error {
	return s.store.SaveEdges(category, s.name, fromUUID, edges)
}

func (s *Transaction) RemoveEdges(category, fromUUID string) error {
	return s.store.RemoveEdges(category, s.name, fromUUID)
}

func (s *Transaction) GetEdges(category, fromUUID string) (edges map[string]string, err error) {
	return s.store.GetEdges(category, s.name, fromUUID)
}

func (s *Transaction) RemoveNode(uuid string) error {
	return s.store.RemoveNode(uuid, s.name)
}

func (s *Transaction) GetNodeProperties(uuid string, requestedProps []string) (props map[string]interface{}, err error) {
	return s.store.GetNodeProperties(uuid, s.name, requestedProps)
}

func (s *Transaction) GetNodeTexts(uuid string, requestedTexts []string) (texts map[string]string, err error) {
	return s.store.GetNodeTexts(uuid, s.name, requestedTexts)
}

func (s *Transaction) GetNodeBlobs(uuid string, requestedBlobs []string, fn func(string, io.Reader) error) error {
	return s.store.GetNodeBlobs(uuid, s.name, requestedBlobs, fn)
}
*/

/*
type Store interface {
	Rollback() error
	SaveNodeProperties(uuid string, props map[string]interface{}) error
	SaveNodeTexts(uuid string, texts map[string]string) error
	SaveNodeBlobs(uuid string, blobs map[string]io.Reader) error
	SaveEdges(category, fromUUID string, edges map[string]string) error
	RemoveEdges(category, fromUUID string) error
	GetEdges(category, fromUUID string) (edges map[string]string, err error)
	RemoveNode(uuid string) error
	GetNodeProperties(uuid string, requestedProps []string) (props map[string]interface{}, err error)
	GetNodeTexts(uuid string, requestedTexts []string) (texts map[string]string, err error)
	GetNodeBlobs(uuid string, requestedBlobs []string, fn func(string, io.Reader) error) error
	Commit(comment string) error
	Shard() string
}
*/

type Transaction interface {
	// only the props that have a key set are going to be changed
	SaveNodeProperties(uuid string, props map[string]interface{}) error

	// map relname => nodeUuid, only the texts that have a key set are going to be changed
	SaveNodeTexts(uuid string, texts map[string]string) error

	// map poolname => []nodeUuid, only the blobs that have a key set are going to be changed
	SaveNodeBlobs(uuid string, blobs map[string]io.Reader) error

	SaveEdges(category, fromUUID string, edges map[string]string) error

	RemoveEdges(category, fromUUID string) error

	GetEdges(category, fromUUID string) (edges map[string]string, err error)

	// remove node with properties, relations and pools
	// references are not checked nor deleted, cascading deletes must be made from the outside
	RemoveNode(uuid string) error

	// only the properties that exist make it into the returned map
	// it is no error if a requested property does not exist for a node
	// the caller has to check the returned map against the requested props if
	// she wants to check, if all requested properties have been returned
	GetNodeProperties(uuid string, requestedProps []string) (props map[string]interface{}, err error)

	// the returned map has as values the uuids of the nodes
	// only the relations that exist make it into the returned map
	// it is no error if a requested relation does not exist for a node
	// the caller has to check the returned map against the requested rels if
	// she wants to check, if all requested relations have been returned
	// also there is no guarantee that the nodes which uuids are returned do still exist.
	// there must be wrappers put around the store to ensure this (preferably by using indices)
	GetNodeTexts(uuid string, requestedTexts []string) (texts map[string]string, err error)

	// the returned map has as values slices of uuids of the nodes
	// only the pools that exist make it into the returned map
	// it is no error if a requested pool does not exist for a node
	// the caller has to check the returned map against the requested pools if
	// she wants to check, if all requested pools have been returned
	// also there is no guarantee that the nodes which uuids are returned do still exist.
	// there must be wrappers put around the store to ensure this (preferably by using indices)
	// GetNodeBlobs(uuid string,  requestedBlobs []string) (pools map[string]io.Reader, err error)
	GetNodeBlobs(uuid string, requestedBlobs []string, fn func(string, io.Reader) error) error
	Shard() string
}

type Store interface {
	Transaction
	// rollback the actions since the last successfull commit
	Rollback() error
	// save the changes in the db
	Commit(comment string) error
}
