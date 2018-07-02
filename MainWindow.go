package main

import (
	"fmt"
	"github.com/carylorrk/goline/api"
	prot "github.com/carylorrk/goline/protocol"
	"sync"
	"time"

	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
)

type MainWindow struct {
	Parent *LoginWindow
	Window *gtk.Window

	Notebook *gtk.Notebook

	FriendsTable    *gtk.Table
	FriendsViewport *gtk.Viewport
	FriendsScroll   *gtk.ScrolledWindow
	FriendsCount    uint

	MoreTable *gtk.Table

	ChatWindows map[string]*ChatWindow

	closeChan  chan bool
	reconnect  uint
	opRevision int64
}

func NewMainWindow(parent *LoginWindow) *MainWindow {
	mainWindow := &MainWindow{Parent: parent}
	mainWindow.ChatWindows = make(map[string]*ChatWindow)
	mainWindow.closeChan = make(chan bool)

	mainWindow.setupUI()
	mainWindow.setupFriendsTable()
	return mainWindow
}

func (self *MainWindow) parseHttpRequest(str string) (code int, err error) {
	_, err = fmt.Sscanf(str, "HTTP Response code: %d", &code)
	return
}

func (self *MainWindow) fetchOperations() []*prot.Operation {
	var operations []*prot.Operation = nil
	var err error
	var clientErr error
	operations, err = goline.client.FetchNewOperations(50)
	if err != nil {
		v, ok := err.(thrift.TTransportException)
		if ok {
			code, parseErr := self.parseHttpRequest(v.Error())
			if parseErr != nil {
				goline.LoggerPrintln("FetchNewOperations:", err, "parseHttpRequest:", parseErr)
				goto errorHandler
			}
			if code == 400 {
				goline.client, clientErr = api.NewLineClient()
				if clientErr != nil {
					goline.LoggerPrintln("FetchNewOperations:", err, "NewLineClient:", clientErr)
					goto errorHandler
				}
				clientErr = goline.client.AuthTokenLogin(goline.AuthToken)
				if clientErr != nil {
					goline.LoggerPrintln("FetchNewOperations:", err, "NewLineClient:", clientErr)
					goto errorHandler
				}
				self.reconnect = 0
			} else if code > 400 {
				goline.LoggerPrintln(err)
				goto errorHandler
			}
		} else {
			goline.LoggerPrintln(err)
			goto errorHandler
		}
	}
	return operations
errorHandler:
	if self.reconnect <= 10 {
		self.reconnect += 1
	} else {
		gdk.ThreadsEnter()
		RunErrorMessage(self.Window, "Failed to get new message! Program closed.")
		gdk.ThreadsLeave()
		gtk.MainQuit()
	}
	return nil
}

func (self *MainWindow) runPoll() {
	for {
		select {
		case <-self.closeChan:
			return
		default:
			operations := self.fetchOperations()
			for _, operation := range operations {
				revision := operation.GetRevision()
				if revision <= self.opRevision {
					continue
				}
				self.reconnect = 0
				message := operation.GetMessage()
				opType := operation.GetTypeA1()
				switch opType {
				case prot.OpType_SEND_MESSAGE:
					fallthrough
				case prot.OpType_SEND_CONTENT:
					fallthrough
				case prot.OpType_RECEIVE_MESSAGE:
					if message != nil {
						if opType == prot.OpType_SEND_MESSAGE &&
							(message.ContentType == prot.ContentType_VIDEO ||
								message.ContentType == prot.ContentType_IMAGE) {
							continue
						}
						if goline.client.Profile == nil {
							var err error
							goline.client, err = api.NewLineClient()
							if err != nil {
								goline.LoggerPrintln(err)
								RunErrorMessage(self.Window, "Failed to get new message! Program closed.")
								gtk.MainQuit()
							}
							err = goline.client.AuthTokenLogin(goline.AuthToken)
							if err != nil {
								goline.LoggerPrintln(err)
								RunErrorMessage(self.Window, "Failed to get new message! Program closed.")
								gtk.MainQuit()
							}
						}
						mid := goline.client.Profile.GetMid()
						fromId := message.GetFrom()
						toId := message.GetTo()
						var id string
						if fromId == mid {
							id = toId
						} else {
							if toId == mid {
								id = fromId
							} else {
								id = toId
							}
						}
						chatWindow := self.ChatWindows[id]
						if chatWindow == nil {
							gdk.ThreadsEnter()
							entity, err := goline.client.GetLineEntityById(id)
							if err != nil {
							}
							if entity != nil {

								self.showChatWindowFactory(entity)()
							}
							gdk.ThreadsLeave()
						} else {
							gdk.ThreadsEnter()
							chatWindow.addSentence(message)
							chatWindow.Conversation.ShowAll()
							gdk.ThreadsLeave()
						}
					}
				}
				self.opRevision = revision
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func (self *MainWindow) refreshFriends() {
	_, err := goline.client.RefreshContacts()
	if err != nil {
		RunErrorMessage(self.Window, "Failed to get new data. No refresh.")
		return
	}
	_, err = goline.client.RefreshGroups()
	if err != nil {
		RunErrorMessage(self.Window, "Failed to get new data. No refresh.")
		return
	}
	_, err = goline.client.RefreshRooms()
	if err != nil {
		RunErrorMessage(self.Window, "Failed to get new data. No refresh.")
		return
	}
	self.FriendsViewport.Remove(self.FriendsTable)
	self.FriendsTable = gtk.NewTable(0, 0, true)
	self.FriendsCount = 0
	self.setupFriendsTable()
	self.FriendsViewport.Add(self.FriendsTable)
	self.FriendsViewport.ShowAll()
}

func (self *MainWindow) setupMoreTab() {
	logout := gtk.NewButtonWithLabel("Logout")
	logout.Clicked(func() {
		goline.AuthToken = ""
		goline.SaveSettings()
		self.Parent.Status = gtk.NewLabel("Please enter your ID and password.")
		self.Parent.Status.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("blue"))
		self.Parent.Login.SetSensitive(true)
		self.closeChan <- true
		self.Parent.Window.ShowAll()
		self.Window.Destroy()
	})
	self.MoreTable.Attach(logout, 0, 1, 0, 1, gtk.FILL|gtk.EXPAND, gtk.FILL, 3, 3)
}

func (self *MainWindow) setupUI() {
	self.Window = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	self.Window.SetTransientFor(self.Parent.Window)
	self.Window.SetPosition(gtk.WIN_POS_CENTER_ON_PARENT)
	self.Window.SetTitle("Goline")
	self.Window.SetDefaultSize(400, 500)
	self.Window.Connect("destroy", func() {
		if self.Parent.Window.GetVisible() == false {
			gtk.MainQuit()
		}
	})

	self.FriendsTable = gtk.NewTable(0, 0, true)
	self.FriendsViewport = gtk.NewViewport(nil, nil)
	self.FriendsViewport.Add(self.FriendsTable)

	self.FriendsScroll = gtk.NewScrolledWindow(nil, nil)
	self.FriendsScroll.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	self.FriendsScroll.Add(self.FriendsViewport)

	self.MoreTable = gtk.NewTable(0, 0, true)
	self.setupMoreTab()

	self.Notebook = gtk.NewNotebook()
	self.Notebook.AppendPage(self.FriendsScroll, gtk.NewLabel("Friends"))
	self.Notebook.AppendPage(self.MoreTable, gtk.NewLabel("More"))

	self.Window.Add(self.Notebook)
}

func (self *MainWindow) FriendsTableAttach(widget gtk.IWidget) {
	self.FriendsTable.Attach(
		widget, 0, 1,
		self.FriendsCount, self.FriendsCount+1,
		gtk.EXPAND|gtk.FILL, gtk.FILL,
		2, 2)
	self.FriendsCount += 1
}

func (self *MainWindow) showChatWindowFactory(entity api.LineEntity) func() {
	var lock sync.Mutex
	return func() {
		lock.Lock()
		chatWindow, chatErr := NewChatWindow(self, entity)
		lock.Unlock()
		switch chatErr {
		case NoError:
			chatWindow.Window.ShowAll()
		case GetMessageBoxError:
		case GetRecentMessagesError:
			RunErrorMessage(self.Window, "Failed to create chat window.")
		}
	}
}

func (self *MainWindow) attachFriend(entity api.LineEntity) {
	btn := gtk.NewButtonWithLabel(entity.GetName())
	btn.Clicked(self.showChatWindowFactory(entity))
	self.FriendsTableAttach(btn)
}

func (self *MainWindow) setupFriendsTable() {
	refresh := gtk.NewButtonWithLabel("Refresh")
	refresh.Clicked(self.refreshFriends)
	self.FriendsTableAttach(refresh)

	self.FriendsTableAttach(gtk.NewLabel("Groups"))
	for _, group := range goline.client.Groups {
		entity := api.NewLineGroupWrapper(group)
		self.attachFriend(entity)
	}

	self.FriendsTableAttach(gtk.NewLabel("Rooms"))
	for _, room := range goline.client.Rooms {
		entity := api.NewLineRoomWrapper(room)
		self.attachFriend(entity)
	}

	self.FriendsTableAttach(gtk.NewLabel("Contacts"))
	for _, contact := range goline.client.Contacts {
		entity := api.NewLineContactWrapper(contact)
		self.attachFriend(entity)
	}
}

func (self *MainWindow) ShowAll() {
	self.Window.ShowAll()
	go self.runPoll()
}
