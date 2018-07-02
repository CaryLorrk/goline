package main

import (
	"github.com/carylorrk/goline/api"
	prot "github.com/carylorrk/goline/protocol"
	"unsafe"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
)

type ChatWindow struct {
	Parent *MainWindow
	Window *gtk.Window

	Table           *gtk.Table
	Conversation    *gtk.Table
	ConversationBox *gtk.EventBox
	Scroll          *gtk.ScrolledWindow
	Input           *gtk.Entry
	Send            *gtk.Button

	Entity       api.LineEntity
	MessageBox   *prot.TMessageBox
	MessageCount uint
}

type ChatWindowError int

const (
	NoError                ChatWindowError = 0
	ChatRoomExist          ChatWindowError = 1
	GetMessageBoxError     ChatWindowError = 2
	GetRecentMessagesError ChatWindowError = 3
)

func NewChatWindow(parent *MainWindow, entity api.LineEntity) (*ChatWindow, ChatWindowError) {
	id := entity.GetId()
	exist := parent.ChatWindows[id]
	if exist != nil {
		return nil, ChatRoomExist

	}

	messageBox, err := goline.client.GetMessageBox(id)
	if err != nil {
		goline.LoggerPrintln(err)
		return nil, GetMessageBoxError
	}

	messages, err := goline.client.GetRecentMessages(messageBox, 20)
	if err != nil {
		goline.LoggerPrintln(err)
		return nil, GetRecentMessagesError
	}

	chatWindow := &ChatWindow{
		Parent:     parent,
		Entity:     entity,
		MessageBox: messageBox}
	chatWindow.setupUI()
	chatWindow.setupWindow()
	chatWindow.setupConversation(messages)
	chatWindow.Input.GrabFocus()
	parent.ChatWindows[id] = chatWindow
	return chatWindow, 0
}

func (self *ChatWindow) sendTextFromInput() {
	text := self.Input.GetText()
	if text != "" {
		goline.client.SendText(self.Entity.GetId(), text)
		self.Input.SetText("")
	}
}

func (self *ChatWindow) setupUI() {
	self.Window = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)

	self.Conversation = gtk.NewTable(0, 0, false)

	self.ConversationBox = gtk.NewEventBox()
	self.ConversationBox.Add(self.Conversation)
	self.ConversationBox.ModifyBG(gtk.STATE_NORMAL, gdk.NewColorRGB(235, 255, 230))

	self.Scroll = gtk.NewScrolledWindow(nil, nil)
	self.Scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	self.Scroll.AddWithViewPort(self.ConversationBox)

	self.Conversation.Connect("size-allocate", func() {
		adj := self.Scroll.GetVAdjustment()
		adj.SetValue(adj.GetUpper() - adj.GetPageSize())
	})

	self.Input = gtk.NewEntry()
	self.Input.Connect("key-press-event", func(ctx *glib.CallbackContext) {
		arg := ctx.Args(0)
		key := *(**gdk.EventKey)(unsafe.Pointer(&arg))
		if key.Keyval == gdk.KEY_Return || key.Keyval == gdk.KEY_KP_Enter {
			self.sendTextFromInput()
		}
	})

	self.Send = gtk.NewButtonWithLabel("Send")
	self.Send.Clicked(func() {
		self.sendTextFromInput()
	})

	self.Table = gtk.NewTable(0, 0, false)
	self.Table.Attach(self.Scroll, 0, 5, 0, 1, gtk.EXPAND|gtk.FILL, gtk.EXPAND|gtk.FILL, 0, 0)
	self.Table.Attach(self.Input, 0, 4, 1, 2, gtk.EXPAND|gtk.FILL, gtk.FILL, 0, 0)
	self.Table.Attach(self.Send, 4, 5, 1, 2, gtk.FILL, gtk.FILL, 0, 0)

	self.Window.Add(self.Table)
}

func (self *ChatWindow) setupWindow() {
	self.Window.SetTitle(self.Entity.GetName())
	self.Window.SetPosition(gtk.WIN_POS_MOUSE)
	self.Window.Resize(400, 500)
	self.Window.Connect("destroy", func() {
		self.Parent.ChatWindows[self.Entity.GetId()] = nil
	})
}

func (self *ChatWindow) addSentence(message *prot.Message) {
	sentence := NewSentence(self, message)
	self.Conversation.Attach(
		sentence.Widget,
		0, 1,
		self.MessageCount, self.MessageCount+1,
		gtk.EXPAND|gtk.FILL, gtk.FILL,
		3, 3)
	self.MessageCount += 1
}

func (self *ChatWindow) setupConversation(messages []*prot.Message) {
	for idx := len(messages) - 1; idx >= 0; idx-- {
		self.addSentence(messages[idx])
	}
}
