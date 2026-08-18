package appwrite

import (
	"os"
	"time"

	"github.com/appwrite/sdk-for-go/v6/appwrite"
	"github.com/appwrite/sdk-for-go/v6/client"
	"github.com/appwrite/sdk-for-go/v6/tablesdb"
)

var (
	appwriteClient client.Client
	tablesDB       *tablesdb.TablesDB
)

func Init() {
	appwriteClient = appwrite.NewClient(
		appwrite.WithEndpoint(os.Getenv("APPWRITE_ENDPOINT")),
		appwrite.WithProject(os.Getenv("APPWRITE_PROJECT_ID")),
		appwrite.WithKey(os.Getenv("APPWRITE_KEY")),
		appwrite.WithTimeout(30*time.Second),
	)

	tablesDB = tablesdb.New(appwriteClient)
}

func GetClient() client.Client {
	return appwriteClient
}

func TablesDB() *tablesdb.TablesDB {
	return tablesDB
}
