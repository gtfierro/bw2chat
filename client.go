package main

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"github.com/gtfierro/ordo/core"
	"github.com/pkg/errors"
	"os"
	"strings"
	"sync"
)

var printRed = color.New(color.FgRed).SprintFunc()
var printYellow = color.New(color.FgYellow).SprintFunc()
var printGreen = color.New(color.FgGreen).SprintFunc()

type OrdoClient struct {
	ordo  *core.OrdoCore
	Alias string

	Input      *bufio.Reader
	Screen     chan core.Message
	roomStates chan core.RoomState

	roomLock    sync.RWMutex
	currentRoom *core.Room

	stopTailing chan bool
}

func NewOrdoClient(entityfile, alias string) *OrdoClient {
	oc := &OrdoClient{
		ordo:        core.NewOrdoCore(entityfile, alias),
		Alias:       alias,
		Input:       bufio.NewReader(os.Stdin),
		Screen:      make(chan core.Message, 100),
		roomStates:  make(chan core.RoomState, 100),
		stopTailing: make(chan bool),
	}

	// display ordo messages on screen
	go func() {
		for msg := range oc.ordo.Log {
			oc.display(msg)
		}
	}()

	return oc
}

func (oc *OrdoClient) display(s string) {
	oc.Screen <- core.Message{
		From:    "<<System>>",
		Message: s,
	}
}

func (oc *OrdoClient) tailRoomState(state core.RoomState) {
	oc.roomStates <- state
}

func (oc *OrdoClient) runCommand(cmd Command) {
	switch cmd.Type {
	case JoinCommand:
		if err := oc.JoinRoom(cmd.Args); err != nil {
			oc.display(printRed("Error joining", err))
		} else {
			oc.display(printGreen("Joined ", cmd.Args[0]))
		}
	case LeaveCommand:
		if err := oc.LeaveRoom(cmd.Args); err != nil {
			oc.display(printRed("Error leaving", err))
		}
	case ListJoinedRoomsCommand:
		rooms := oc.ordo.GetRooms()
		if len(rooms) > 0 {
			tmp := "Joined Rooms:\n"
			for _, room := range rooms {
				tmp += fmt.Sprintf("%s\n", room.URI)
			}
			tmp += "\n---\n"
			oc.display(printYellow(tmp))
		} else {
			oc.display(printYellow("No rooms joined"))
		}
	//case ERRORCommand:
	//	cc.display(printRed(fmt.Sprintf("ERROR: unrecognized command: %+v", cmd)))
	case HelpCommand:
		oc.display(printYellow("\\join <uri> -- join the room if you have permission"))
		oc.display(printYellow("\\leave -- Leaves the room, but also causes echoing characters to stop BUGGY DO NOT USE"))
		oc.display(printYellow("\\listjoined -- Lists the rooms you have joined"))
		oc.display(printYellow("\\help-- Prints this help"))
	default:
		oc.SendMessage(strings.Join(cmd.Args, " "))
	}
}

func (oc *OrdoClient) JoinRoom(args []string) error {
	var (
		roomURI string // args 0
	)

	if len(args) < 1 {
		return errors.New("Need 1 argument to JoinRoom")
	}
	roomURI = args[0]

	oc.roomLock.Lock()
	defer oc.roomLock.Unlock()

	if oc.currentRoom != nil && oc.currentRoom.URI == roomURI {
		oc.display(printYellow("Already in room ", roomURI))
		return nil
	}

	room, err := oc.ordo.JoinRoom(roomURI)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Could not join room %s", roomURI))
	}
	if oc.currentRoom != nil {
		oc.currentRoom.StopTail()
	}
	room.StartTail(oc.Screen)
	room.SetStateUpdateCallback(oc.tailRoomState)
	oc.currentRoom = room
	return nil
}

func (oc *OrdoClient) LeaveRoom(args []string) error {
	oc.roomLock.Lock()
	defer oc.roomLock.Unlock()
	if oc.currentRoom == nil {
		oc.display(printYellow("Not in a room to leave"))
		return nil
	}
	var reason string
	if len(args) == 0 {
		reason = "<No reason given>"
	} else {
		reason = args[0]
	}
	err := oc.currentRoom.Leave(reason)
	oc.currentRoom = nil
	return err
}

func (oc *OrdoClient) SendMessage(msg string) {
	oc.roomLock.Lock()
	defer oc.roomLock.Unlock()
	if oc.currentRoom == nil {
		oc.display(printYellow("Must join room first: \\join <roomuri>"))
		return
	}
	oc.currentRoom.Speak(msg)
}
