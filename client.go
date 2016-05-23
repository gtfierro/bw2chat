package main

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	_ "github.com/pkg/errors"
	bw "gopkg.in/immesys/bw2bind.v3"
	"os"
	"strings"
	"sync"
)

var printRed = color.New(color.FgRed).SprintFunc()
var printYellow = color.New(color.FgYellow).SprintFunc()
var printGreen = color.New(color.FgGreen).SprintFunc()

type ChatClient struct {
	C          *bw.BW2Client
	vk         string
	Namespace  string
	Alias      string
	Input      *bufio.Reader
	Screen     chan ChatMessage
	roomStates chan roomState

	currentRoomLock sync.RWMutex
	CurrentRoom     *ChatRoom

	roomLock    sync.RWMutex
	JoinedRooms map[string]*ChatRoom

	stopTailing chan bool
}

func NewChatClient(entityfile, namespace, alias string) *ChatClient {
	cc := &ChatClient{
		C:           bw.ConnectOrExit(""),
		Namespace:   namespace,
		Alias:       alias,
		Input:       bufio.NewReader(os.Stdin),
		Screen:      make(chan ChatMessage, 10),
		JoinedRooms: make(map[string]*ChatRoom),
		roomStates:  make(chan roomState, 100),
		stopTailing: make(chan bool),
	}

	cc.vk = cc.C.SetEntityFileOrExit(entityfile)
	cc.C.OverrideAutoChainTo(true) //TODO: what does this do again?

	return cc
}

func (cc *ChatClient) buildURI(suffix string) string {
	return cc.Namespace + suffix
}

func (cc *ChatClient) buildRoomURI(room *ChatRoom) string {
	return cc.Namespace + "room/" + room.Name
}

func (cc *ChatClient) display(s string) {
	cc.Screen <- ChatMessage{
		Room:    "!!System!!",
		From:    "!!System!!",
		Message: s,
	}
}

func (cc *ChatClient) runCommand(cmd Command) {
	switch cmd.Type {
	case JoinCommand:
		if err := cc.JoinRoom(cmd.Args[0]); err != nil {
			cc.display(printRed("Error joining", err))
		}
	case LeaveCommand:
		cc.display(printYellow("Running leave"))
		if err := cc.LeaveRoom(); err != nil {
			cc.display(printRed("Error leaving", err))
		}
	case ListJoinedRoomsCommand:
		cc.roomLock.RLock()
		rooms := []string{}
		for room, _ := range cc.JoinedRooms {
			rooms = append(rooms, room)
		}
		cc.roomLock.RUnlock()
		if len(rooms) > 0 {
			tmp := "Joined Rooms:\n"
			tmp += strings.Join(rooms, "\n")
			tmp += "\n---\n"
			cc.display(printYellow(tmp))
		} else {
			cc.display(printYellow("No rooms joined"))
		}
	case ERRORCommand:
		cc.display(printRed(fmt.Sprintf("ERROR: unrecognized command: %+v", cmd)))
	case HelpCommand:
		cc.display(printYellow("\\join <roomname> -- join the room if you have permission"))
		cc.display(printYellow("\\leave -- Leaves the room, but also causes echoing characters to stop BUGGY DO NOT USE"))
		cc.display(printYellow("\\listjoined -- Lists the rooms you have joined"))
		cc.display(printYellow("\\help-- Prints this help"))
	default:
		cc.currentRoomLock.Lock()
		defer cc.currentRoomLock.Unlock()
		if cc.CurrentRoom == nil {
			cc.display(printYellow("Must join room first: \\join <roomname>"))
			return
		}
		message := &ChatMessage{
			Room:    cc.CurrentRoom.Name,
			From:    cc.Alias,
			Message: strings.Join(cmd.Args, " "),
		}
		cc.SendMessage(message)
	}
}

// joins the chat room. returns error if the room doesn't exist
func (cc *ChatClient) JoinRoom(roomname string) error {
	var (
		err     error
		room    *ChatRoom
		found   bool
		newRoom = false
	)
	cc.roomLock.Lock()

	cc.currentRoomLock.Lock()
	defer cc.currentRoomLock.Unlock()

	if room, found = cc.JoinedRooms[roomname]; !found {
		room, err = NewChatRoom(roomname, cc, ChatRoomBufSize)
		if err != nil {
			cc.roomLock.Unlock()
			return err
		}
		cc.display(printGreen(fmt.Sprintf("Joined room %s", roomname)))
		cc.JoinedRooms[roomname] = room
		go func() {
			for state := range room.state {
				cc.roomStates <- state
			}
		}()
		newRoom = true
	} else if cc.CurrentRoom == room {
		cc.display(printGreen(fmt.Sprintf("Already joined room %s", roomname)))
	} else {
		cc.display(printGreen(fmt.Sprintf("Joining room %s", roomname)))
		newRoom = true
	}

	if newRoom {
		if cc.CurrentRoom != nil {
			cc.stopTailing <- true
		}
		cc.CurrentRoom = room
		go func() {
			for {
				select {
				case <-cc.stopTailing:
					cc.display(printYellow("Unfocusing ", roomname))
					return
				case msg := <-room.Buffer:
					cc.Screen <- msg
					room.readMessage()
				}
			}
		}()
	}

	cc.roomLock.Unlock()

	return nil
}

func (cc *ChatClient) LeaveRoom() error {
	leaveRoom := LeaveRoom{Alias: cc.Alias}
	if cc.CurrentRoom != nil && cc.CurrentRoom.Buffer != nil {
		close(cc.CurrentRoom.Buffer)
	}
	err := cc.C.Publish(&bw.PublishParams{
		URI:            cc.buildRoomURI(cc.CurrentRoom),
		PayloadObjects: []bw.PayloadObject{leaveRoom.ToBW()},
	})
	if err != nil {
		cc.display(printRed(err.Error()))
		return err
	}
	return nil
}

// create the chat room if it doesn't exist and join it
func (cc *ChatClient) CreateAndJoin(roomname string) error {
	roomToCreate := CreateRoom{Name: roomname}
	err := cc.C.Publish(&bw.PublishParams{
		URI:            cc.buildURI(CreateRoomTopic),
		PayloadObjects: []bw.PayloadObject{roomToCreate.ToBW()},
	})
	cc.JoinRoom(roomname)
	return err
}

func (cc *ChatClient) SendMessage(msg *ChatMessage) {
	err := cc.C.Publish(&bw.PublishParams{
		URI:            cc.buildRoomURI(cc.CurrentRoom),
		PayloadObjects: []bw.PayloadObject{msg.ToBW()},
	})
	if err != nil {
		cc.display(printRed(err.Error()))
	}
}
