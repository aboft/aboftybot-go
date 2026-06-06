package utils

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	as "github.com/aerospike/aerospike-client-go/v8"
)

func lineCountKey(channel string, date string) string {
	return fmt.Sprintf("%s|%s", channel, date)
}

func GetLineCount(conn *as.Client, channel string, date string) string {
	lineDate := time.Now().Format("2006-01-02")

	if date != "" {
		d, err := time.Parse("2006-01-02", date)
		if err != nil {
			return "Unable to parse date provided; use YYYY-MM-DD"
		}
		lineDate = d.Format("2006-01-02")
	}

	key, err := as.NewKey("aboftybot", "line_counts", lineCountKey(channel, lineDate))
	if err != nil {
		log.Println("Unable to create line count key:", err)
		return "Unable to retrieve line count. Blame aboft."
	}

	record, err := conn.Get(nil, key, "count")
	if err != nil || record == nil {
		return fmt.Sprintf("No lines found for %s on %s.", channel, lineDate)
	}

	return fmt.Sprintf("There were %v lines said in %s on %s.", record.Bins["count"], channel, lineDate)
}

func IncrementLineCount(conn *as.Client, channel string) {
	now := time.Now().Format("2006-01-02")

	writePolicy := as.NewWritePolicy(0, 0)
	writePolicy.SendKey = true

	key, err := as.NewKey("aboftybot", "line_counts", lineCountKey(channel, now))
	if err != nil {
		log.Println("Unable to create line count key:", err)
		return
	}

	_, err = conn.Operate(
		writePolicy,
		key,
		as.PutOp(as.NewBin("channel", channel)),
		as.PutOp(as.NewBin("date", now)),
		as.AddOp(as.NewBin("count", 1)),
	)

	if err != nil {
		log.Println("Unable to increment line count:", err)
	}
}

func GetTopLineCounts(conn *as.Client, channel string) string {
	stmt := as.NewStatement("aboftybot", "line_counts")
	stmt.SetFilter(as.NewEqualFilter("channel", channel))

	recordSet, err := conn.Query(nil, stmt)
	if err != nil {
		log.Println("Unable to retrieve top line counts:", err)
		return "Unable to retrieve top line counts."
	}
	defer recordSet.Close()

	type lineCount struct {
		date  string
		count int
	}

	recordMap := make([]lineCount, 0, 500)

	for result := range recordSet.Results() {
		if result.Err != nil {
			log.Println("Query Error:", result.Err)
			continue
		}

		rec := result.Record
		if rec == nil {
			continue
		}

		count, ok := rec.Bins["count"].(int)
		if !ok {
			continue
		}

		pk := rec.Key.Value().String()
		_, date, found := strings.Cut(pk, "|")
		if !found {
			date = pk // fallback for old records
		}

		recordMap = append(recordMap, lineCount{
			date:  date,
			count: count,
		})
	}

	if len(recordMap) == 0 {
		return "Unable to retrieve top line counts."
	}

	sort.Slice(recordMap, func(i, j int) bool {
		return recordMap[i].count > recordMap[j].count
	})

	limit := 5
	if len(recordMap) < limit {
		limit = len(recordMap)
	}

	var b strings.Builder
	for i := 0; i < limit; i++ {
		fmt.Fprintf(&b, "%s: %d, ", recordMap[i].date, recordMap[i].count)
	}

	return strings.TrimSuffix(b.String(), ", ")
}

func GetLastNDaysLineCounts(conn *as.Client, channel string, days string) string {
	daysInt, err := strconv.Atoi(days)
	if err != nil || daysInt <= 0 {
		return "Invalid number of days: " + days
	}

	if daysInt > 15 {
		daysInt = 15
	}

	keys := make([]*as.Key, 0, daysInt)
	labels := make([]string, 0, daysInt)

	for i := 0; i < daysInt; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		pk := fmt.Sprintf("%s|%s", channel, date)

		key, _ := as.NewKey("aboftybot", "line_counts", pk)
		keys = append(keys, key)
		labels = append(labels, date)
	}

	recs, err := conn.BatchGet(nil, keys, "count")
	if err != nil {
		log.Println("BatchGet error:", err)
		return "Unable to retrieve line counts. Blame aboft."
	}

	var b strings.Builder

	for i, rec := range recs {
		if rec == nil {
			fmt.Fprintf(&b, "%s: 0, ", labels[i])
			continue
		}

		count, _ := rec.Bins["count"].(int)
		fmt.Fprintf(&b, "%s: %d, ", labels[i], count)
	}

	resp := strings.TrimSuffix(b.String(), ", ")
	if resp == "" {
		return "No line counts found."
	}

	return resp
}
