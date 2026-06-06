package main

import (
	"reflect"
	"testing"
)

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name string
		line string
		want ircMessage
	}{
		{
			name: "ping with trailing",
			line: "PING :tmi.host\r\n",
			want: ircMessage{command: "PING", trailing: "tmi.host"},
		},
		{
			name: "full privmsg",
			line: ":nick!user@host PRIVMSG #chan :hello world\r\n",
			want: ircMessage{
				prefix:   "nick!user@host",
				command:  "PRIVMSG",
				params:   []string{"#chan"},
				trailing: "hello world",
			},
		},
		{
			name: "numeric welcome command",
			line: ":liberty.snoonet.org 001 aboftybot-dev :Welcome to the Snoonet IRC Network\r\n",
			want: ircMessage{
				prefix:   "liberty.snoonet.org",
				command:  "001",
				params:   []string{"aboftybot-dev"},
				trailing: "Welcome to the Snoonet IRC Network",
			},
		},
		{
			name: "prefix only",
			line: ":server.name\r\n",
			want: ircMessage{prefix: "server.name"},
		},
		{
			name: "empty line",
			line: "\r\n",
			want: ircMessage{},
		},
		{
			name: "command with params but no trailing",
			line: ":nick!user@host JOIN #chan\r\n",
			want: ircMessage{
				prefix:  "nick!user@host",
				command: "JOIN",
				params:  []string{"#chan"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMessage(tt.line)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseMessage(%q) = %+v, want %+v", tt.line, got, tt.want)
			}
		})
	}
}

func TestParsePastLinesCommand(t *testing.T) {
	msg := parseMessage(":aboft!aboft@user/aboft PRIVMSG #aboftybot-dev :.pastl 2\r\n")

	if msg.trailing != ".pastl 2" {
		t.Fatalf("trailing = %q", msg.trailing)
	}
}
