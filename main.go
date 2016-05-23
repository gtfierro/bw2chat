package main

//TODO: want persistent storage for chatroom stuff
//TODO: call bw.SilenceLog
import (
	"github.com/codegangsta/cli"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	bw "gopkg.in/immesys/bw2bind.v3"
	"os"
)

const VERSION = "0.0.1"
const ChatRoomBufSize = 200 // buffer 200 messages

var (
	CreateRoomTopic = "create"
	log             = logging.MustGetLogger("bw2chat_daemon")
	format          = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
)

func init() {
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

type ChatDaemon struct {
	C         *bw.BW2Client
	vk        string
	Namespace string

	stop chan bool

	newRooms chan *bw.SimpleMessage
}

func NewChatDaemon(entityfile, namespace string) *ChatDaemon {
	cd := &ChatDaemon{
		C:         bw.ConnectOrExit(""),
		Namespace: namespace,
		stop:      make(chan bool),
	}
	cd.vk = cd.C.SetEntityFileOrExit(entityfile)
	cd.C.OverrideAutoChainTo(true) //TODO: what does this do again?

	return cd
}

func (cd *ChatDaemon) buildURI(suffix string) string {
	return cd.Namespace + suffix
}

func (cd *ChatDaemon) Start() {
	var err error
	// subscribe to the CreateRoom topic
	if cd.newRooms, err = cd.C.Subscribe(&bw.SubscribeParams{URI: cd.buildURI(CreateRoomTopic)}); err != nil {
		log.Fatal(errors.Wrap(err, "Could not subscribe to Create Rooms"))
	}

	go func() {
		log.Notice("Listening for new rooms...")
		var createRoomMsg CreateRoom
		for msg := range cd.newRooms {
			for _, po := range msg.POs {
				if po.IsType(CreateRoomTopicPID, CreateRoomTopicPID) {
					err := po.(bw.MsgPackPayloadObject).ValueInto(&createRoomMsg)
					if err != nil {
						log.Error(errors.Wrap(err, "Could not parse create room msg"))
					} else {
						log.Infof("Got create room msg: %+v", createRoomMsg)
					}
				}
			}
		}
	}()

	<-cd.stop
}

func startDaemon(c *cli.Context) {
	daemon := NewChatDaemon(c.GlobalString("entity"), c.GlobalString("namespace"))
	daemon.Start()
}

func startClient(c *cli.Context) {
	client := NewChatClient(c.GlobalString("entity"), c.GlobalString("namespace"), c.String("alias"))
	//client.CreateAndJoin(c.String("room"))
	StartUserInterface(client)
	//client.repl()
	x := make(chan bool)
	<-x
}

func main() {
	app := cli.NewApp()
	app.Name = "bw2chat"
	app.Version = VERSION

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "entity,e",
			Value: "chatroomtest.ent",
			Usage: "The entity to use when interacting w/ bw2chat",
		},
		cli.StringFlag{
			Name:  "namespace,n",
			Value: "gabe.ns/chatrooms/",
			Usage: "Root namespace for chatrooms",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "daemon",
			Usage:  "Chat daemon maintains state for chatrooms",
			Action: startDaemon,
		},
		{
			Name:   "client",
			Usage:  "Chat client",
			Action: startClient,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "room",
					Value: "default",
					Usage: "Chat room to join",
				},
				cli.StringFlag{
					Name:  "alias,nickname",
					Value: "jf_sebastian",
					Usage: "Nickname to use",
				},
			},
		},
	}

	//daemon := NewChatDaemon("chatroomtest.ent", "gabe.ns/chatrooms/")
	app.Run(os.Args)
}
