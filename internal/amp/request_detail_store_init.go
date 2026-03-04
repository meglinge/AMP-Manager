package amp

import "ampmanager/internal/database"

func init() {
	getDBPathFunc = database.GetPath
}
