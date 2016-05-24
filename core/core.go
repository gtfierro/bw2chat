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
		Log:   make(chan string, 100),
		bw:    bw.ConnectOrExit(""),
		rooms: make(map[string]*Room),
	}
	ordo.vk = ordo.bw.SetEntityFileOrExit(entityfile)
	ordo.bw.OverrideAutoChainTo(true)
	ordo.Alias = alias
	ordo.log(ordo.Alias)

	return ordo
}

func (ordo *OrdoCore) log(s string) {
	select {
	case ordo.Log <- s:
	default:
		<-ordo.Log
		ordo.Log <- s
	}
}

// fills in the map with our current rooms
func (ordo *OrdoCore) GetRooms() []*Room {
	ordo.roomsLock.Lock()
	defer ordo.roomsLock.Unlock()
	r := []*Room{}
	for _, room := range ordo.rooms {
		r = append(r, room)
	}
	return r
}

// Join the chatroom at the given URI using alias as your nickname. Needs consume privileges to
// listen in the room, and publish privileges to send messages to the room.
// Returns true if the room was created for the first time
func (ordo *OrdoCore) JoinRoom(roomURI string) (*Room, error) {
	var (
		room  *Room
		found bool
		err   error
	)
	ordo.roomsLock.Lock()
	if room, found = ordo.rooms[roomURI]; !found {
		// need to join the room
		if room, err = NewRoom(roomURI, ordo, RoomBufSize); err != nil {
			ordo.log(fmt.Sprintf("Could not join room (%s)", err.Error()))
			return nil, err
		}
		ordo.rooms[roomURI] = room
	}

	if !room.Alive {
		ordo.log(fmt.Sprintf("room %s not alive so joining", room.URI))
		if err = ordo.performJoin(room); err != nil {
			ordo.log(fmt.Sprintf("Could not join room %s at URI %s (%s)", room.Name, room.URI, err.Error()))
			return nil, err
		}
	}
	ordo.log(fmt.Sprintf("Joined room %s at URI %s", room.Name, room.URI))

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
	message := &ChatMessage{Alias: ordo.Alias, Message: msg}
	err := ordo.bw.Publish(&bw.PublishParams{
		URI:            room.URI,
		PayloadObjects: []bw.PayloadObject{message.ToBW()},
	})
	if err != nil {
		ordo.log(fmt.Sprintf("Could not send to room %s at URI %s (%s)", room.Name, room.URI, err.Error()))
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
		ordo.log(fmt.Sprintf("Could not send Leave to room %s at URI %s (%s)", room.Name, room.URI, err.Error()))
		return err
	}
	return nil
}
