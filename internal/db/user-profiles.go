package db

type UserProfileData struct {
	Name string
}

const SchemaUserProfiles = `
CREATE TABLE IF NOT EXISTS user_profiles (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL
);`

func (db *DB) CreateUserProfile(e execer, name string) (int64, error) {
	res, err := e.Exec(`INSERT INTO user_profiles (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	e.Emit(DBEvent{
		Table: "user_profiles",
		Type:  DBCreate,
		Data:  nil,
	})

	return id, nil
}
