package zoom

import "errors"

var ErrNoCommit = errors.New("do not commit")

// Transaction executes a transaction on the given store
// each action is a function that receives a store and returns an error
// most of the time that functions will be the Save() and Remove() methods of a Node
// but also arbitrary code may be used as function (use methods to share state
// between the actions)
// the actions are invoked one after another and if one of them returns an error, the execution
// stops and the transaction is rolled back and the error is returned
// if err is not nil and rolledback=false then you're in trouble:
// the transaction did not complete and the rollback wasn't successfull either
// func (s *Shard) Transaction(comment string, actions ...func(Store) error) (rolledback bool, err error) {
func NewTransaction(st Store, comment CommitMessage, action func(t Transaction) error) (err error) {
	// for _, a := range actions {
	err = action(st)
	if err == ErrNoCommit {
		st.Rollback()
		return nil
	}
	if err != nil {
		// rollBackErr := store.Rollback()
		// rolledback = rollBackErr == nil
		st.Rollback()
		return
	}
	// }

	err = st.Commit(comment)
	if err != nil {
		// rollBackErr := store.Rollback()
		// rolledback = rollBackErr == nil
		st.Rollback()
	}
	return
}
