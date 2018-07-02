package main

import (
	"github.com/carylorrk/goline/api"
	prot "github.com/carylorrk/goline/protocol"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
)

type LoginWindow struct {
	Window *gtk.Window

	IdLabel *gtk.Label
	IdEntry *gtk.Entry

	PasswdLabel *gtk.Label
	PasswdEntry *gtk.Entry

	Status   *gtk.Label
	Remember *gtk.CheckButton
	Login    *gtk.Button
	Exit     *gtk.Button

	Table *gtk.Table

	DataChan chan string
	ErrChan  chan error
}

func NewLoginWindow() *LoginWindow {
	loginWindow := &LoginWindow{}
	loginWindow.DataChan = make(chan string)
	loginWindow.ErrChan = make(chan error)
	loginWindow.setupUI()
	return loginWindow
}

func (self *LoginWindow) CheckAuthToken() {
	if goline.AuthToken != "" {
		go func() {
			var err error
			gdk.ThreadsEnter()
			self.Status.SetText("Login with previous authorization token...")
			self.Status.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("blue"))
			self.Login.SetSensitive(false)
			gdk.ThreadsLeave()

			goline.client, err = api.NewLineClient()
			if err != nil {
				goto errorHandler
			}

			err = goline.client.AuthTokenLogin(goline.AuthToken)
			if err != nil {
				goto errorHandler
			}

			gdk.ThreadsEnter()
			NewMainWindow(self).ShowAll()
			self.Window.Hide()
			gdk.ThreadsLeave()
			return

		errorHandler:
			goline.LoggerPrintln(err)
			gdk.ThreadsEnter()
			RunAlertMessage(self.Window, "Failed to login with previous authorization token.")
			self.Status.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("red"))
			self.Status.SetText("Faild to login with previous authorization token")
			self.Login.SetSensitive(true)
			gdk.ThreadsLeave()
			goline.AuthToken = ""
			err = goline.SaveSettings()
			if err != nil {
				goline.LoggerPrintln(err)
				RunAlertMessage(self.Window, "Failed to clean previous token in settings file.")
			}
			return

		}()
	}

}

func (self *LoginWindow) setupUI() {
	self.Window = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	self.Window.SetPosition(gtk.WIN_POS_CENTER)
	self.Window.SetTitle("Goline - Login")
	self.Window.Resize(400, 500)
	self.Window.Connect("destroy", gtk.MainQuit)

	self.IdLabel = gtk.NewLabel("ID")
	self.IdLabel.SetAlignment(0, 0.5)

	self.IdEntry = gtk.NewEntry()
	self.IdEntry.SetText(goline.Id)

	self.PasswdLabel = gtk.NewLabel("Password")
	self.PasswdLabel.SetAlignment(0, 0.5)

	self.PasswdEntry = gtk.NewEntry()
	self.PasswdEntry.SetText(goline.Password)
	self.PasswdEntry.SetInvisibleChar('*')
	self.PasswdEntry.SetVisibility(false)

	self.Status = gtk.NewLabel("Please enter your ID and password.")
	self.Status.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("blue"))
	self.Status.SetAlignment(0, 0.5)
	self.Remember = gtk.NewCheckButtonWithLabel("Remember your ID and password")
	self.Remember.SetActive(goline.Remember)

	self.Login = gtk.NewButtonWithLabel("Login")
	self.Login.Clicked(self.newLoginClickedCallback())

	self.Exit = gtk.NewButtonWithLabel("Exit")
	self.Exit.Clicked(func() {
		self.Window.Emit("destroy")
	})

	self.Table = gtk.NewTable(6, 4, true)
	self.Table.Attach(self.IdLabel, 0, 1, 0, 1, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)
	self.Table.Attach(self.IdEntry, 1, 4, 0, 1, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)
	self.Table.Attach(self.PasswdLabel, 0, 1, 1, 2, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)
	self.Table.Attach(self.PasswdEntry, 1, 4, 1, 2, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)
	self.Table.Attach(self.Status, 0, 4, 2, 3, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)
	self.Table.Attach(self.Remember, 0, 4, 3, 4, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)
	self.Table.Attach(self.Login, 0, 4, 4, 5, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)
	self.Table.Attach(self.Exit, 0, 4, 5, 6, gtk.EXPAND|gtk.FILL, gtk.FILL, 5, 5)

	self.Window.Add(self.Table)
}

func (self *LoginWindow) newLoginClickedCallback() func() {
	return func() {
		err := self.updateSettings()
		if err != nil {
			goline.LoggerPrintln(err)
			RunAlertMessage(self.Window, "Failed to save settings.")
		}

		pincode, err := self.getPincode()
		if err != nil {
			goline.LoggerPrintln(err)
			RunErrorMessage(self.Window, "Failed to get pincode.")
			return
		}

		self.verify(pincode)

	}
}

func (self *LoginWindow) updateSettings() error {
	if self.Remember.GetActive() {
		goline.Id = self.IdEntry.GetText()
		goline.Password = self.PasswdEntry.GetText()
	} else {
		goline.Id = ""
		goline.Password = ""
	}
	goline.Remember = self.Remember.GetActive()
	err := goline.SaveSettings()
	return err
}

func (self *LoginWindow) getPincode() (string, error) {
	var err error
	goline.client, err = api.NewLineClient()
	if err != nil {
		goline.LoggerPrintln(err)
		return "", err
	}

	go func() {
		pincode, err := goline.client.GetPincode(goline.Id, goline.Password)
		self.ErrChan <- err
		self.DataChan <- pincode
	}()
	self.Status.SetText("Get verification code...")
	self.Status.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("blue"))
	err = <-self.ErrChan
	pincode := <-self.DataChan
	if err != nil {
		goline.LoggerPrintln(err)
		var reason string
		switch v := err.(type) {
		case *prot.TalkException:
			reason = v.GetReason()
		default:
			reason = "Ooooops, something went wrong!"
		}
		self.Status.SetText(reason)
		return "", err
	}
	return pincode, nil
}

func (self *LoginWindow) verify(pincode string) {
	verificationWindow := NewVerificationWindow(self, pincode)
	verificationWindow.Window.ShowAll()
	self.Window.Hide()

	go func() {
		authToken, err := goline.client.GetAuthTokenAfterVerify()
		self.ErrChan <- err
		self.DataChan <- authToken
	}()

	go func() {
		err := <-self.ErrChan
		authToken := <-self.DataChan
		if err != nil {
			gdk.ThreadsEnter()
			self.Status.SetText(err.Error())
			self.Status.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("red"))
			gdk.ThreadsLeave()
			if err.Error() != "Cancel login." {
				goline.LoggerPrintln(err)
				verificationWindow.Window.Emit("destroy")
			} else {
				self.Status.ModifyFG(gtk.STATE_NORMAL, gdk.NewColorRGB(255, 255, 0))
			}
			return
		}
		gdk.ThreadsEnter()
		self.Status.SetText("Login...")
		self.Status.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("blue"))
		verificationWindow.Window.Emit("destroy")
		gdk.ThreadsLeave()

		goline.AuthToken = authToken
		goline.SaveSettings()
		if err != nil {
			goline.LoggerPrintln(err)
			RunAlertMessage(self.Window, "Failed to save new token.")
		}
		err = goline.client.AuthTokenLogin(authToken)
		if err != nil {
			goline.LoggerPrintln(err)
			gdk.ThreadsEnter()
			RunErrorMessage(self.Window, "Failed to login with authentication token.")
			self.Status.SetText("Failed to login with authentication token.")
			self.Status.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("red"))
			gdk.ThreadsLeave()
			return
		}

		gdk.ThreadsEnter()
		NewMainWindow(self).ShowAll()
		self.Window.Hide()
		gdk.ThreadsLeave()
	}()
}
