package utils

import (
	"fmt"
	"log"
	"os"

	as "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-client-go/v8/types"
)

func StartDB(db_host string) *as.Client {
	clientPolicy := as.NewClientPolicy()
	clientPolicy.AuthMode = as.AuthModeInternal
	clientPolicy.User = os.Getenv("DB_USER")
	clientPolicy.Password = os.Getenv("DB_PASS")
	clientPolicy.MinConnectionsPerNode = 5

	client, err := as.NewClientWithPolicy(clientPolicy, db_host, 3000)

	if err != nil {
		log.Println("Unable to connect to Aerospike:", err)
	}

	log.Println("Client connected!")

	return client
}

func CreateSecondaryIndex(conn *as.Client) error {
	idxTask, err := conn.CreateIndex(
		nil,
		"aboftybot",
		"line_counts",
		"lineCountIdx",
		"channel",
		as.STRING,
	)
	if err != nil {
		if err.Matches(types.INDEX_FOUND) {
			log.Println("Secondary index already exists")
			return nil
		}
		return fmt.Errorf("create secondary index: %w", err)
	}

	if err := <-idxTask.OnComplete(); err != nil {
		return fmt.Errorf("wait for secondary index: %w", err)
	}

	log.Println("Secondary index created")

	return nil
}
