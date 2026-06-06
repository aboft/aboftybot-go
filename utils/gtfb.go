package utils

import (
	"log"
	"math/rand"
	"time"

	as "github.com/aerospike/aerospike-client-go/v8"
)

func GetInsult(client *as.Client) string {

	queryPolicy := as.NewQueryPolicy()
	queryPolicy.MaxRetries = 0
	queryPolicy.TotalTimeout = 1000 * time.Millisecond

	stmt := as.NewStatement("aboftybot", "gtfb", "insult")

	records, err := client.Query(queryPolicy, stmt)

	if err != nil {
		log.Println("Unable to get insults from bar.gtfb: ", err)
	}

	var insults []string

	for record := range records.Results() {
		if record.Err != nil {
			log.Println("Unable to get insults from bar.gtfb: ", record.Err)
		} else {
			if insult, ok := record.Record.Bins["insult"].(string); ok {
				insults = append(insults, insult)
			}
		}
	}

	if len(insults) > 0 {
		rand.Seed(time.Now().UnixNano())
		min := 0
		max := len(insults) - 1
		insultIdx := rand.Intn(max-min+1) + min
		return insults[insultIdx]
	}

	return "Unable to find insults. Blame aboft."
}
