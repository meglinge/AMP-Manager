package database

import "database/sql"

func prepareStandaloneDatabase(options Options) (*sql.DB, error) {
	normalized, err := options.Normalize()
	if err != nil {
		return nil, err
	}

	mu.Lock()
	defer mu.Unlock()

	previousDB := db
	previousPath := dbPath
	previousType := dbType
	previousOptions := dbOptions
	previousInited := inited

	db = nil
	dbPath = ""
	dbType = ""
	dbOptions = Options{}
	inited = false

	if err := initDB(normalized); err != nil {
		if db != nil {
			_ = db.Close()
		}
		db = previousDB
		dbPath = previousPath
		dbType = previousType
		dbOptions = previousOptions
		inited = previousInited
		return nil, err
	}

	preparedDB := db

	db = previousDB
	dbPath = previousPath
	dbType = previousType
	dbOptions = previousOptions
	inited = previousInited

	return preparedDB, nil
}
