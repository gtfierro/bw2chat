package main

import (
	"strings"
)

/*
Here we implement a very very basic parser for some chatroom commands.
We simplify the IRC model a little bit: all commands begin with '\'

/join <roomname> -- joins a room

<just text w/o a command> -- text to send to chat room


*/

type CommandType uint8

const (
	JoinCommand CommandType = iota + 1
	LeaveCommand
	SendCommand
	ListJoinedRoomsCommand
	HelpCommand
	ERRORCommand
)

func (ct CommandType) String() string {
	switch ct {
	case JoinCommand:
		return "Join"
	case LeaveCommand:
		return "Leave"
	case SendCommand:
		return "Send"
	case ListJoinedRoomsCommand:
		return "ListJoinedRooms"
	case HelpCommand:
		return "Help"
	case ERRORCommand:
		return "Error"
	default:
		return "Unknown"
	}
}

func commandFromString(s string) (c CommandType) {
	switch strings.TrimSpace(s[1:]) {
	case "join":
		c = JoinCommand
	case "leave":
		c = LeaveCommand
	case "listjoined":
		c = ListJoinedRoomsCommand
	case "help":
		c = HelpCommand
	default:
		c = ERRORCommand
	}
	return
}

type Command struct {
	Type CommandType
	Args []string
}

func Parse(input string) Command {
	cmd := Command{}
	s := strings.TrimSpace(input)

	if strings.HasPrefix(s, "\\") {
		tmp := strings.SplitAfterN(s, " ", -1)
		cmd.Type = commandFromString(tmp[0])
		cmd.Args = tmp[1:]
	} else {
		cmd.Type = SendCommand
		cmd.Args = []string{s}
	}

	return cmd
}
