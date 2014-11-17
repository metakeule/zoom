package zoom

// Transaction executes a transaction on the given store
// each action is a function that receives a store and returns an error
// most of the time that functions will be the Save() and Remove() methods of a Node
// but also arbitrary code may be used as function (use methods to share state
// between the actions)
// the actions are invoked one after another and if one of them returns an error, the execution
// stops and the transaction is rolled back and the error is returned
// if err is not nil and rolledback=false then you're in trouble:
// the transaction did not complete and the rollback wasn't successfull either
func Transaction(store Store, comment string, actions ...func(Store) error) (rolledback bool, err error) {
	for _, a := range actions {
		err = a(store)
		if err != nil {
			rollBackErr := store.Rollback()
			rolledback = rollBackErr == nil
			return
		}
	}

	err = store.Commit(comment)
	if err != nil {
		rollBackErr := store.Rollback()
		rolledback = rollBackErr == nil
	}
	return
}
