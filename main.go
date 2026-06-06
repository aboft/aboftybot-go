package main

import (
	"bufio"
	"fmt"
	"gobot/utils"
	"log"
	"net"
	"os"
	"strings"

	as "github.com/aerospike/aerospike-client-go/v8"
)

type ircMessage struct {
	prefix   string
	command  string
	params   []string
	trailing string
}

func sendRaw(conn net.Conn, format string, args ...any) {
	_, err := fmt.Fprintf(conn, format+"\r\n", args...)
	if err != nil {
		log.Println("write error:", err)
	}
}

func sendPrivmsg(conn net.Conn, channel, msg string) {
	sendRaw(conn, "PRIVMSG %s :%s", channel, msg)
}

func main() {
	// Check for required environment variables
	required := []string{"IRC_SERVER", "IRC_NICK", "IRC_PASSWORD", "IRC_CHANNEL", "DB_HOST"}
	for _, key := range required {
		if os.Getenv(key) == "" {
			log.Fatalf("missing required env var: %s", key)
		}
	}

	// Connect to IRC server and fail fast if it fails
	// “fail fast” means if a required dependency is unavailable at startup,
	// exit immediately instead of continuing in a broken state.
	conn, err := net.Dial("tcp", os.Getenv("IRC_SERVER"))
	if err != nil {
		log.Fatalf("IRC dial failed: %v", err)
	}
	defer conn.Close()

	dbConn := utils.StartDB(os.Getenv("DB_HOST"))
	if err := utils.CreateSecondaryIndex(dbConn); err != nil {
		log.Fatalf("create secondary index: %v", err)
	}

	reader := bufio.NewReader(conn)

	conn.Write([]byte("NICK " + os.Getenv("IRC_NICK") + " \r\n"))
	conn.Write([]byte("USER " + os.Getenv("IRC_NICK") + " * localhost " + os.Getenv("IRC_NICK") + " \r\n"))

	joined := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading from IRC: %v", err)
		}
		log.Println(line)
		msg := parseMessage(line)

		switch msg.command {
		case "PING":
			sendRaw(conn, "PONG %s", msg.trailing)
		case "001":
			sendRaw(conn, "JOIN %s", os.Getenv("IRC_CHANNEL"))
		case "INVITE":
			sendRaw(conn, "JOIN %s", msg.trailing)
		case "PRIVMSG":
			parsePrivmsg(msg, conn, dbConn)
		case "396":
			sendPrivmsg(conn, "nickserv", "identify "+os.Getenv("IRC_PASSWORD"))
			if !joined {
				sendRaw(conn, "JOIN %s", os.Getenv("IRC_CHANNEL"))
			}
		case "JOIN":
			if msg.prefix == os.Getenv("IRC_CHANNEL") {
				joined = true
			}
		}

	}
}

func parseMessage(line string) ircMessage {
	line = strings.TrimRight(line, "\r\n")

	msg := ircMessage{}

	if strings.HasPrefix(line, ":") {
		parts := strings.SplitN(line[1:], " ", 2)
		msg.prefix = parts[0]
		if len(parts) == 1 {
			return msg
		}
		line = parts[1]
	}

	if i := strings.Index(line, " :"); i != -1 {
		msg.trailing = line[i+2:]
		line = line[:i]
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return msg
	}

	msg.command = parts[0]
	if len(parts) > 1 {
		msg.params = parts[1:]
	}

	return msg
}

func parsePrivmsg(msg ircMessage, conn net.Conn, dbConn *as.Client) {
	if len(msg.params) == 0 {
		return
	}

	channel := msg.params[0]
	user := strings.Split(msg.prefix, "!")[0]

	parts := strings.Fields(msg.trailing)
	if len(parts) == 0 {
		utils.IncrementLineCount(dbConn, channel)
		return
	}

	command := parts[0]
	args := parts[1:]

	utils.IncrementLineCount(dbConn, channel)

	switch command {
	case ".go":
		target := user
		if len(args) > 0 {
			target = args[0]
		}
		sendPrivmsg(conn, channel, "Go. Be gone. Away from me "+target+".")
	case ".lines":
		date := ""
		if len(args) > 0 {
			date = args[0]
		}
		lineCount := utils.GetLineCount(dbConn, channel, date)
		sendPrivmsg(conn, channel, lineCount)
	case ".gtfb":
		target := user
		if len(args) > 0 {
			target = args[0]
		}
		insult := utils.GetInsult(dbConn)
		sendPrivmsg(conn, channel, fmt.Sprintf("(%s): %s.", target, insult))
	case ".topl":
		topLineCounts := utils.GetTopLineCounts(dbConn, channel)
		sendPrivmsg(conn, channel, fmt.Sprintf("(%s) Top Lines: %s.", user, topLineCounts))
	case ".pastl":
		days := "7"
		if len(args) > 0 {
			days = args[0]
		}
		lastNDaysLineCounts := utils.GetLastNDaysLineCounts(dbConn, channel, days)
		sendPrivmsg(conn, channel, fmt.Sprintf("(%s) Last %s Days: %s.", user, days, lastNDaysLineCounts))
	}
}
