package core

import (
	"fmt"
	"github.com/pkg/errors"
	bw "gopkg.in/immesys/bw2bind.v5"
	"strings"
	"sync/atomic"
)

// represents an Ordo chat room
type Room struct {
	// buffer of received but not displayed messages
	buffer chan Message
	// URI of the room
	URI string
	// name of room derived from URI
	Name string
	// your name in the room
	Alias string
	// whether or not the room can be used
	Alive bool
	// internal signal to leave the room
	quit chan bool
	// internal signal to stop tailing
	stoptail chan bool

	// callback when the room's state updates
	updateState func(state RoomState)

	// number of unread messages
	unreadMsgCount int32
	// number of known users
	knownUserCount int32
	// map of known user VKs to aliases
	knownUsers map[string]string

	// reference to core
	ordo *OrdoCore
	// channel of incoming chat room messages
	subscription chan *bw.SimpleMessage
}

func NewRoom(roomURI string, ordo *OrdoCore, bufsize int) (*Room, error) {
	room := &Room{
		buffer:      make(chan Message, bufsize),
		URI:         roomURI,
		Alive:       false,
		stoptail:    make(chan bool, 1),
		quit:        make(chan bool),
		updateState: func(state RoomState) {},
		knownUsers:  map[string]string{ordo.vk: ordo.Alias},
		ordo:        ordo,
	}
	if idx := strings.LastIndex(roomURI, "/"); idx > 0 {
		room.Name = roomURI[strings.LastIndex(roomURI, "/"):]
	} else {
		return nil, errors.New(fmt.Sprintf("Invalid URI %s", roomURI))
	}
	return room, nil
}

// function to be invoked when the room's state changes
func (room *Room) SetStateUpdateCallback(fxn func(state RoomState)) {
	room.updateState = fxn
}

// join the room. Sends a JoinMessage to all subscribers and
// subscribes to the room
func (room *Room) Join() error {
	if !room.Alive {
		return room.ordo.performJoin(room)
	}
	return nil
}

func (room *Room) Leave(reason string) error {
	room.Alive = false
	if err := room.ordo.performLeave(room, reason); err != nil {
		return err
	}
	room.quit <- true
	close(room.subscription)
	close(room.buffer)
	return nil
}

func (room *Room) Speak(msg string) error {
	return room.ordo.performSpeak(room, msg)
}

func (room *Room) StartTail(dest chan Message) {
	go func(dest chan Message) {
		for {
			select {
			case <-room.stoptail:
				return
			case msg := <-room.buffer:
				dest <- Message{From: msg.From, Room: room, Message: msg.Message}
				atomic.AddInt32(&room.unreadMsgCount, -1)
				room.getState()
			}
		}
	}(dest)
}

func (room *Room) StopTail() {
	room.stoptail <- true
}

func (room *Room) newMessage(msg Message) {
	if !room.Alive {
		return
	}
	select {
	case room.buffer <- msg:
		atomic.AddInt32(&room.unreadMsgCount, 1)
		room.getState()
	default:
		<-room.buffer
		room.buffer <- msg
		atomic.AddInt32(&room.unreadMsgCount, 1)
	}
}

func (room *Room) listen() {
	go func() {
		var (
			chatMessage  ChatMessage
			joinMessage  JoinRoom
			leaveMessage LeaveRoom
		)
		for {
			select {
			case <-room.quit:
				return
			case msg := <-room.subscription:
				for _, po := range msg.POs {
					if po.IsType(ChatMessagePID, ChatMessagePID) {
						err := po.(bw.MsgPackPayloadObject).ValueInto(&chatMessage)
						if err != nil {
							log.Error(errors.Wrap(err, "Could not parse chat msg"))
						}
						if len(chatMessage.Message) == 0 {
							continue
						}
						if _, found := room.knownUsers[msg.From]; !found {
							room.knownUsers[msg.From] = chatMessage.Alias
						}
						room.newMessage(Message{
							Message: chatMessage.Message,
							FromVK:  msg.From,
							From:    room.knownUsers[msg.From],
							Room:    room,
						})
						room.getState()
					} else if po.IsType(JoinRoomPID, JoinRoomPID) {
						err := po.(bw.MsgPackPayloadObject).ValueInto(&joinMessage)
						if err != nil {
							log.Error(errors.Wrap(err, "Could not parse join msg"))
						}
						room.knownUsers[msg.From] = joinMessage.Alias
						atomic.AddInt32(&room.knownUserCount, 1)
						room.getState()
					} else if po.IsType(LeaveRoomPID, LeaveRoomPID) {
						err := po.(bw.MsgPackPayloadObject).ValueInto(&leaveMessage)
						if err != nil {
							log.Error(errors.Wrap(err, "Could not parse leave msg"))
						}
						delete(room.knownUsers, msg.From)
						atomic.AddInt32(&room.knownUserCount, -1)
						room.getState()
					}
				}
			}
		}
	}()
}

type RoomState struct {
	NumUnreadMessages int32
	NumCurrentUsers   int32
	Name              string
	CurrentUsers      map[string]string
	Room              *Room
}

func (state RoomState) Height() int {
	return 5 + len(state.CurrentUsers)
}

func (room *Room) getState() {
	room.updateState(RoomState{
		NumUnreadMessages: atomic.LoadInt32(&room.unreadMsgCount),
		NumCurrentUsers:   atomic.LoadInt32(&room.knownUserCount),
		Name:              room.Name,
		CurrentUsers:      room.knownUsers,
		Room:              room,
	})
}
