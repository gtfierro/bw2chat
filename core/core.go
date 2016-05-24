package core

import (
	"fmt"
	"github.com/op/go-logging"
	bw "gopkg.in/immesys/bw2bind.v5"
	"os"
	"sync"
)

const (
	RoomBufSize = 1000
)

var (
	log    = logging.MustGetLogger("ordocore")
	format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
)

func init() {
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

type OrdoCore struct {
	// connection to bosswave
	bw *bw.BW2Client
	// verifying key
	vk string

	roomsLock sync.RWMutex
	rooms     map[string]*Room

	// log of actions taken
	Log chan string
	// your name
	Alias string

	// handlers
	ReceivedJoin  func(msg JoinRoom)
	ReceivedLeave func(msg LeaveRoom)
	ReceivedChat  func(msg ChatMessage)
}

func NewOrdoCore(entityfile, alias string) *OrdoCore {
	ordo := &OrdoCore{
		bw: bw.ConnectOrExit(""),
	}
	ordo.vk = ordo.bw.SetEntityFileOrExit(entityfile)
	ordo.bw.OverrideAutoChainTo(true)

	return ordo
}

// Join the chatroom at the given URI using alias as your nickname. Needs consume privileges to
// listen in the room, and publish privileges to send messages to the room
func (ordo *OrdoCore) JoinRoom(roomURI string) (*Room, error) {
	var (
		room  *Room
		found bool
	)
	ordo.roomsLock.Lock()
	if room, found = ordo.rooms[roomURI]; !found {
		// need to join the room
		room = NewRoom(roomURI, ordo, RoomBufSize)
	}

	if err := ordo.performJoin(room); err != nil {
		ordo.Log <- fmt.Sprintf("Could not join room %s at URI %s (%s)", room.Name, room.URI, err.Error())
		return nil, err
	}
	ordo.Log <- fmt.Sprintf("Joined room %s at URI %s", room.Name, room.URI)

	ordo.roomsLock.Unlock()

	return room, nil
}

func (ordo *OrdoCore) performJoin(room *Room) error {
	joinRoom := JoinRoom{Alias: ordo.Alias}
	err := ordo.bw.Publish(&bw.PublishParams{
		URI:            room.URI,
		PayloadObjects: []bw.PayloadObject{joinRoom.ToBW()},
	})
	if err != nil {
		return err
	}
	room.subscription, err = ordo.bw.Subscribe(&bw.SubscribeParams{
		URI: room.URI,
	})
	if err != nil {
		return err
	}
	room.listen()
	room.Alive = true
	return nil
}

func (ordo *OrdoCore) performSpeak(room *Room, msg string) error {
	message := &ChatMessage{msg}
	err := ordo.bw.Publish(&bw.PublishParams{
		URI:            room.URI,
		PayloadObjects: []bw.PayloadObject{message.ToBW()},
	})
	if err != nil {
		ordo.Log <- fmt.Sprintf("Could not send to room %s at URI %s (%s)", room.Name, room.URI, err.Error())
		return err
	}
	return nil
}

func (ordo *OrdoCore) performLeave(room *Room, reason string) error {
	msg := &LeaveRoom{Reason: reason}
	err := ordo.bw.Publish(&bw.PublishParams{
		URI:            room.URI,
		PayloadObjects: []bw.PayloadObject{msg.ToBW()},
	})
	if err != nil {
		ordo.Log <- fmt.Sprintf("Could not send Leave to room %s at URI %s (%s)", room.Name, room.URI, err.Error())
		return err
	}
	return nil
}
