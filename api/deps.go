package api

import "go.etcd.io/bbolt"

var apiDB *bbolt.DB

func setDB(db *bbolt.DB) {
	apiDB = db
}

func getDB() *bbolt.DB {
	if apiDB == nil {
		panic("api database dependency not initialized")
	}
	return apiDB
}
