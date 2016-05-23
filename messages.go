package main

import (
	bw "gopkg.in/immesys/bw2bind.v3"
)

const (
	CreateRoomTopicPIDString = "2.0.7.1"
	ChatMessagePIDString     = "2.0.7.2"
	JoinRoomPIDString        = "2.0.7.3"
	LeaveRoomPIDString       = "2.0.7.4"
)

var (
	CreateRoomTopicPID = bw.FromDotForm("2.0.7.1")
	ChatMessagePID     = bw.FromDotForm("2.0.7.2")
	JoinRoomPID        = bw.FromDotForm("2.0.7.3")
	LeaveRoomPID       = bw.FromDotForm("2.0.7.4")
)

// messages for chat room
type CreateRoom struct {
	Name string
}

func (msg CreateRoom) ToBW() bw.PayloadObject {
	po, _ := bw.CreateMsgPackPayloadObject(CreateRoomTopicPID, msg)
	return po
}

type ChatMessage struct {
	Room string
	From string
	//TODO: any public key?
	Message string
}

func (msg ChatMessage) ToBW() bw.PayloadObject {
	po, _ := bw.CreateMsgPackPayloadObject(ChatMessagePID, msg)
	return po
}

func WrapError(err error) *ChatMessage {
	return &ChatMessage{
		Room:    "local",
		From:    "local",
		Message: err.Error(),
	}
}

type JoinRoom struct {
	Alias string
}

func (msg JoinRoom) ToBW() bw.PayloadObject {
	po, _ := bw.CreateMsgPackPayloadObject(JoinRoomPID, msg)
	return po
}

type LeaveRoom struct {
	Alias string
}

func (msg LeaveRoom) ToBW() bw.PayloadObject {
	po, _ := bw.CreateMsgPackPayloadObject(LeaveRoomPID, msg)
	return po
}
