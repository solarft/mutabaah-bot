package appwrite

import (
	"os"

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
	)
	/*
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				var lastErr error
				for _, server := range []string{"1.1.1.1:53", "8.8.8.8:53"} {
					d := net.Dialer{Timeout: 5 * time.Second}
					conn, err := d.DialContext(ctx, network, server)
					if err == nil {
						return conn, nil
					}
					lastErr = err
				}
				return nil, lastErr
			},
		}

		appwriteClient.Client = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
					Resolver:  resolver,
				}).DialContext,
				ForceAttemptHTTP2:   true,
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		}
	*/

	tablesDB = tablesdb.New(appwriteClient)
}

func GetClient() client.Client {
	return appwriteClient
}

func TablesDB() *tablesdb.TablesDB {
	return tablesDB
}
