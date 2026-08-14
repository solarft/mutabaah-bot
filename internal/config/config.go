package config

import "os"

var (
	DatabaseID   string
	UsersTableID string
)

func Init() {
	DatabaseID = os.Getenv("DATABASE_ID")
	if DatabaseID == "" {
		panic("DATABASE_ID is not set!")
	}

	UsersTableID = os.Getenv("USERS_TABLE_ID")
	if UsersTableID == "" {
		panic("USERS_TABLE_ID is not set!")
	}
}
