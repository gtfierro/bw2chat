package main

import (
	"github.com/pkg/errors"
	bw "gopkg.in/immesys/bw2bind.v3"
	"sync/atomic"
)

type ChatRoom struct {
	Buffer             chan ChatMessage
	UnreadMessageCount int32
	CurrentUsersCount  int32
	Name               string
	Users              map[string]bool
	client             *ChatClient
	subscription       chan *bw.SimpleMessage
	Namespace          string
	state              chan roomState
}

func NewChatRoom(roomname string, client *ChatClient, bufsize int) (*ChatRoom, error) {
	room := &ChatRoom{
		Name:               roomname,
		Buffer:             make(chan ChatMessage, bufsize),
		Users:              make(map[string]bool),
		client:             client,
		Namespace:          client.Namespace,
		UnreadMessageCount: 0,
		CurrentUsersCount:  0,
		state:              make(chan roomState, 50),
	}

	joinRoom := JoinRoom{Alias: client.Alias}
	err := client.C.Publish(&bw.PublishParams{
		URI:            room.buildRoomURI(),
		PayloadObjects: []bw.PayloadObject{joinRoom.ToBW()},
	})
	if err != nil {
		return room, err
	}

	room.subscription, err = client.C.Subscribe(&bw.SubscribeParams{
		URI: room.buildRoomURI(),
	})
	if err != nil {
		return room, err
	}

	room.listen()
	room.getState()

	return room, nil
}

func (room *ChatRoom) newMessage(msg ChatMessage) {
	select {
	case room.Buffer <- msg:
		atomic.AddInt32(&room.UnreadMessageCount, 1)
	default:
		<-room.Buffer
		room.Buffer <- msg
	}
}

func (room *ChatRoom) readMessage() {
	atomic.AddInt32(&room.UnreadMessageCount, -1)
	room.getState()
}

func (room *ChatRoom) buildRoomURI() string {
	return room.Namespace + "room/" + room.Name
}

func (room *ChatRoom) listen() {
	go func() {
		var (
			chatMessage  ChatMessage
			joinMessage  JoinRoom
			leaveMessage LeaveRoom
		)
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
					room.Users[joinMessage.Alias] = true
					atomic.AddInt32(&room.CurrentUsersCount, 1)
					room.getState()
				} else if po.IsType(LeaveRoomPID, LeaveRoomPID) {
					err := po.(bw.MsgPackPayloadObject).ValueInto(&leaveMessage)
					if err != nil {
						log.Error(errors.Wrap(err, "Could not parse leave msg"))
					}
					delete(room.Users, leaveMessage.Alias)
					atomic.AddInt32(&room.CurrentUsersCount, -1)
					room.getState()
				}
			}
		}
	}()
}

type roomState struct {
	NumUnreadMessages int32
	NumCurrentUsers   int32
	FullName          string
	CurrentUsers      map[string]bool
}

func (state roomState) height() int {
	return 5 + len(state.CurrentUsers)
}

func (room *ChatRoom) getState() {
	room.state <- roomState{
		NumUnreadMessages: atomic.LoadInt32(&room.UnreadMessageCount),
		NumCurrentUsers:   atomic.LoadInt32(&room.CurrentUsersCount),
		FullName:          room.Name,
		CurrentUsers:      room.Users,
	}
}
