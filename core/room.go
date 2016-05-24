package core

import (
	"github.com/pkg/errors"
	bw "gopkg.in/immesys/bw2bind.v5"
	"strings"
	"sync/atomic"
)

// represents an Ordo chat room
type Room struct {
	// buffer of received but not displayed messages
	Buffer chan ChatMessage
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

	// callback when the room's state updates
	updateState func(state roomState)

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

func NewRoom(roomURI string, ordo *OrdoCore, bufsize int) *Room {
	room := &Room{
		Buffer:      make(chan ChatMessage, bufsize),
		URI:         roomURI,
		Name:        roomURI[strings.LastIndex(roomURI, "/"):],
		Alive:       false,
		quit:        make(chan bool),
		updateState: func(state roomState) {},
		ordo:        ordo,
	}
	return room
}

// function to be invoked when the room's state changes
func (room *Room) SetStateUpdateCallback(fxn func(state roomState)) {
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
	close(room.Buffer)
	return nil
}

func (room *Room) Speak(msg string) error {
	return room.ordo.performSpeak(room, msg)
}

func (room *Room) newMessage(msg ChatMessage) {
	if !room.Alive {
		return
	}
	select {
	case room.Buffer <- msg:
		atomic.AddInt32(&room.unreadMsgCount, 1)
	default:
		<-room.Buffer
		room.Buffer <- msg
	}
}

func (room *Room) listen() {
	go func() {
		var (
			chatMessage  ChatMessage
			joinMessage  JoinRoom
			leaveMessage LeaveRoom
		)
		select {
		case <-room.quit:
			return
		default:
		}
		for msg := range room.subscription {
			for _, po := range msg.POs {
				if po.IsType(ChatMessagePID, ChatMessagePID) {
					err := po.(bw.MsgPackPayloadObject).ValueInto(&chatMessage)
					if err != nil {
						log.Error(errors.Wrap(err, "Could not parse chat msg"))
					}
					room.newMessage(chatMessage)
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
	}()
}

type roomState struct {
	NumUnreadMessages int32
	NumCurrentUsers   int32
	Name              string
	CurrentUsers      map[string]string
}

func (state roomState) height() int {
	return 5 + len(state.CurrentUsers)
}

func (room *Room) getState() {
	room.updateState(roomState{
		NumUnreadMessages: atomic.LoadInt32(&room.unreadMsgCount),
		NumCurrentUsers:   atomic.LoadInt32(&room.knownUserCount),
		Name:              room.Name,
		CurrentUsers:      room.knownUsers,
	})
}
