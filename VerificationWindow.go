package main

import (
	"errors"
	"github.com/mattn/go-gtk/gtk"
)

type VerificationWindow struct {
	Parent *LoginWindow
	Window *gtk.Window

	Title   *gtk.Label
	Content *gtk.Label
	Code    *gtk.Label
	Cancel  *gtk.Button

	Table *gtk.Table

	Pincode string
}

func NewVerificationWindow(parent *LoginWindow, pincode string) *VerificationWindow {
	verificationWindow := &VerificationWindow{}
	verificationWindow.Parent = parent
	verificationWindow.Pincode = pincode
	verificationWindow.setupUI()

	return verificationWindow
}

func (self *VerificationWindow) setupUI() {

	self.Window = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	self.Window.SetTransientFor(self.Parent.Window)
	self.Window.SetTitle("Goline - Verifiacation")
	self.Window.SetPosition(gtk.WIN_POS_CENTER_ON_PARENT)
	self.Window.Resize(400, 500)
	self.Window.Connect("destroy", func() {
		self.Parent.Window.ShowAll()
	})

	self.Title = gtk.NewLabel("Verify Your Account")
	self.Content = gtk.NewLabel("Please enter the verification code below into your mobile device.")
	self.Code = gtk.NewLabel(self.Pincode)

	self.Cancel = gtk.NewButtonWithLabel("Cancel")
	self.Cancel.Clicked(func() {
		self.Parent.ErrChan <- errors.New("Cancel login.")
		self.Window.Destroy()
	})

	self.Table = gtk.NewTable(4, 1, false)
	self.Table.Attach(self.Title, 0, 1, 0, 1, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)
	self.Table.Attach(self.Content, 0, 1, 1, 2, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)
	self.Table.Attach(self.Code, 0, 1, 2, 3, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)
	self.Table.Attach(self.Cancel, 0, 1, 3, 4, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)

	self.Window.Add(self.Table)

}
