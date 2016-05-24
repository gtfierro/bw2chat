package core

import (
	bw "gopkg.in/immesys/bw2bind.v5"
)

const (
	ChatMessagePIDString = "2.0.7.2"
	JoinRoomPIDString    = "2.0.7.3"
	LeaveRoomPIDString   = "2.0.7.4"
)

var (
	ChatMessagePID = bw.FromDotForm("2.0.7.2")
	JoinRoomPID    = bw.FromDotForm("2.0.7.3")
	LeaveRoomPID   = bw.FromDotForm("2.0.7.4")
)

type ChatMessage struct {
	// the message to send to the chatroom
	Message string
}

func (msg ChatMessage) ToBW() bw.PayloadObject {
	po, _ := bw.CreateMsgPackPayloadObject(ChatMessagePID, msg)
	return po
}

func WrapError(err error) *ChatMessage {
	return &ChatMessage{
		Message: err.Error(),
	}
}

type JoinRoom struct {
	// the name you will be known by in the chatroom
	Alias string
}

func (msg JoinRoom) ToBW() bw.PayloadObject {
	po, _ := bw.CreateMsgPackPayloadObject(JoinRoomPID, msg)
	return po
}

type LeaveRoom struct {
	// why you left the chatroom. Will be sent to all members in the room
	Reason string
}

func (msg LeaveRoom) ToBW() bw.PayloadObject {
	po, _ := bw.CreateMsgPackPayloadObject(LeaveRoomPID, msg)
	return po
}
