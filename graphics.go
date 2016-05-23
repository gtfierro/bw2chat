package main

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/pkg/errors"
)

type UserInterface struct {
	g          *gocui.Gui
	client     *ChatClient
	header     string
	roomOffset int
}

func StartUserInterface(client *ChatClient) *UserInterface {
	ui := &UserInterface{
		g:          gocui.NewGui(),
		client:     client,
		header:     fmt.Sprintf("[%s]> ", client.Alias),
		roomOffset: -1,
	}

	if err := ui.g.Init(); err != nil {
		log.Fatal(errors.Wrap(err, "Could not initialize terminal interface"))
	}
	ui.g.SetLayout(ui.layout)
	if err := ui.keybindings(ui.g); err != nil {
		log.Fatal(errors.Wrap(err, "Could not assign key bindings"))
	}

	go func() {
		defer ui.g.Close()
		if err := ui.g.MainLoop(); err != nil && err != gocui.ErrQuit {
			log.Fatal(errors.Wrap(err, "Main loop of gocui broke"))
		}
	}()

	go func() {
		for state := range ui.client.roomStates {
			ui.g.Execute(func(g *gocui.Gui) error {
				v, err := g.View(state.FullName)
				if err == gocui.ErrUnknownView {
					v, err = g.SetView(state.FullName, -1, ui.roomOffset, 30, ui.roomOffset+state.height())
					ui.roomOffset += state.height()
					if err != gocui.ErrUnknownView {
						return err
					}
				} else if err != nil {
					return err
				}

				v.Clear()
				fmt.Fprintln(v, fmt.Sprintf("Room: [%s]", state.FullName))
				fmt.Fprintln(v, fmt.Sprintf("  Unread(%d)", state.NumUnreadMessages))
				fmt.Fprintln(v, fmt.Sprintf("  Users(%d)", state.NumCurrentUsers))
				for user, ok := range state.CurrentUsers {
					if ok {
						fmt.Fprintln(v, fmt.Sprintf("    %s", user))
					}
				}
				return nil
			})
		}
	}()

	return ui
}

func (ui *UserInterface) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	// sidebar
	if v, err := g.SetView("sidebar", -1, -1, 30, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = true
		fmt.Fprintln(v, "ROOMS")
	}
	// chatroom header
	if v, err := g.SetView("chatroomname", 30, -1, maxX, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = true
		v.FgColor = gocui.AttrBold
		fmt.Fprintln(v, "<No Chatroom Joined>")
	}
	// chatroom
	if v, err := g.SetView("chatroom", 30, 2, maxX, maxY-10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = true
		v.Autoscroll = true
		go func() {
			for msg := range ui.client.Screen {
				from := msg.From
				message := msg.Message
				g.Execute(func(g *gocui.Gui) error {
					v, err := g.View("chatroom")
					if err != nil {
						log.Fatal(errors.Wrap(err, "Could not update chatroom screen"))
					}
					fmt.Fprintln(v, fmt.Sprintf("[%s]> %s", from, message))
					return nil
				})
			}
		}()

	}
	// input prompt
	if v, err := g.SetView("prompt", 30, maxY-10, 30+len(ui.header), maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		fmt.Fprint(v, ui.header)
	}

	// input box
	if v, err := g.SetView("input", 30+len(ui.header), maxY-10, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		g.Cursor = true
		v.Wrap = true
		v.Editable = true
		v.Frame = false
		if err := g.SetCurrentView("input"); err != nil {
			return err
		}
		//v.SetCursor(len(ui.header), 0)
	}

	return nil
}

func (ui *UserInterface) keybindings(g *gocui.Gui) error {
	if err := ui.g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, ui.quit); err != nil {
		log.Fatal(err)
	}
	if err := ui.g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, ui.parse); err != nil {
		log.Fatal(err)
	}
	return nil
}

func (ui *UserInterface) quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (ui *UserInterface) parse(g *gocui.Gui, v *gocui.View) error {
	input := v.ViewBuffer()
	if len(input) == 0 {
		v.Clear()
		v.SetCursor(0, 0)
		return nil
	}
	cmd := Parse(input[:len(input)-1])

	// clear input
	v.Clear()
	v.SetCursor(0, 0)

	go ui.client.runCommand(cmd)

	switch cmd.Type {
	case JoinCommand:
		g.Execute(func(g *gocui.Gui) error {
			v, err := g.View("chatroomname")
			if err != nil {
				log.Fatal(errors.Wrap(err, "Could not update input screen"))
			}
			v.Clear()
			fmt.Fprintln(v, "Chatroom: ", cmd.Args[0])
			fmt.Fprintln(v, "URI: ", ui.client.Namespace+cmd.Args[0])
			return nil
		})
	case LeaveCommand:
		g.Execute(func(g *gocui.Gui) error {
			v, err := g.View("chatroomname")
			if err != nil {
				log.Fatal(errors.Wrap(err, "Could not update input screen"))
			}
			v.Clear()
			fmt.Fprintln(v, "Chatroom: None")
			fmt.Fprintln(v, "URI: None")
			return nil
		})
	}
	return nil
}
