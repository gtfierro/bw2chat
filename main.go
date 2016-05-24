package main

//TODO: want persistent storage for chatroom stuff
//TODO: call bw.SilenceLog
import (
	"github.com/codegangsta/cli"
	"github.com/op/go-logging"
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

func startDaemon(c *cli.Context) {
	//daemon := NewChatDaemon(c.GlobalString("entity"), c.GlobalString("namespace"))
	//daemon.Start()
}

func startClient(c *cli.Context) {
	client := NewOrdoClient(c.GlobalString("entity"), c.String("alias"))
	StartUserInterface(client)
	for _, room := range c.StringSlice("room") {
		client.runCommand(Command{Type: JoinCommand, Args: []string{room}})
	}
	//TODO: exit cleanly here
	x := make(chan bool)
	<-x
}

func main() {
	app := cli.NewApp()
	app.Name = "bw2chat"
	app.Version = VERSION

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "entity,e",
			Value:  "chatroomtest.ent",
			Usage:  "The entity to use when interacting w/ bw2chat",
			EnvVar: "BW2_DEFAULT_ENTITY",
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
					Name:  "alias,nickname",
					Value: "jf_sebastian",
					Usage: "Nickname to use",
				},
				cli.StringSliceFlag{
					Name:  "room, r",
					Value: &cli.StringSlice{},
					Usage: "List of rooms to join on startup. Use a new -r for each room",
				},
			},
		},
	}

	app.Run(os.Args)
}
