package main

import (
	"github.com/carylorrk/goline/api"
	prot "github.com/carylorrk/goline/protocol"
	"path"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
	"github.com/mattn/go-gtk/pango"
)

type Sentence struct {
	Parent  *ChatWindow
	Message *prot.Message

	Widget gtk.IWidget
}

func NewSentence(parent *ChatWindow, message *prot.Message) *Sentence {
	sentence := &Sentence{Parent: parent, Message: message}
	sentence.setupWidget()
	return sentence
}

func (self *Sentence) setupWidget() {
	contentType := self.Message.GetContentType()
	//TODO: Support MIME type
	switch contentType {
	case prot.ContentType_NONE:
		self.handleText(self.Message.GetText(), nil)
	case prot.ContentType_IMAGE:
		self.handleImage()
	case prot.ContentType_VIDEO:
		self.handleVideo()
	case prot.ContentType_AUDIO:
		self.handleAudio()
	case prot.ContentType_STICKER:
		self.handleSticker()
	default:
		self.handleText(contentType.String(), gdk.NewColor("red"))
	}
}

func (self *Sentence) handleSticker() {
	meta := self.Message.ContentMetadata
	stkid := meta["STKID"]
	stkpkgid := meta["STKPKGID"]
	stkver := meta["STKVER"]
	url := api.LINE_STICKER_URL + stkver + "/" + stkpkgid + "/PC/stickers/" + stkid + ".png"
	filePath := path.Join(goline.TempDirPath, "sticker", stkid+".png")
	if CheckFileNotExist(filePath) {
		err := DownloadFile(url, filePath)
		if err != nil {
			goline.LoggerPrintln(err)
			self.handleText("Failed to download sticker.", gdk.NewColor("red"))
			return

		}

	}
	image := gtk.NewImageFromFile(filePath)
	self.Widget = self.tableLayout(image)
}

func (self *Sentence) tableLayout(widget gtk.IWidget) gtk.IWidget {
	table := gtk.NewTable(0, 0, false)
	fromId := self.Message.GetFrom()
	space := gtk.NewLabel("")

	if fromId == goline.client.Profile.GetMid() {
		table.Attach(space, 0, 1, 0, 1, gtk.FILL|gtk.EXPAND, gtk.FILL, 0, 0)
		table.Attach(widget, 1, 2, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
		return table
	} else {
		table.Attach(widget, 0, 1, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
		table.Attach(space, 1, 2, 0, 1, gtk.FILL|gtk.EXPAND, gtk.FILL, 0, 0)

		name := self.getNameById(fromId)
		label := gtk.NewLabel(name + ": ")
		label.ModifyFG(gtk.STATE_NORMAL, self.NewRandomColorFromId(fromId))
		label.SetAlignment(0, 0)

		nameTable := gtk.NewTable(2, 1, false)
		nameTable.Attach(label, 0, 1, 0, 1, gtk.FILL, gtk.FILL, 0, 3)
		nameTable.Attach(table, 1, 2, 0, 1, gtk.FILL, gtk.FILL, 0, 3)
		return nameTable
	}
}

func (self *Sentence) handleAudio() {
	messageId := self.Message.GetId()
	label := gtk.NewLabel("Download Audio")
	box := gtk.NewEventBox()
	box.Add(label)
	box.SetEvents(int(gdk.BUTTON_RELEASE_MASK))
	box.Connect("button-release-event", func() {
		go self.showDownloadWindow(api.LINE_OBJECT_STORAGE_URL + messageId)
	})
	self.Widget = self.tableLayout(box)

}

func (self *Sentence) handleVideo() {
	messageId := self.Message.GetId()
	previewFilePath := path.Join(goline.TempDirPath, "preview", messageId)
	if CheckFileNotExist(previewFilePath) {
		previewUrl := api.LINE_OBJECT_STORAGE_URL + messageId + "/preview"
		err := DownloadFile(previewUrl, previewFilePath)
		if err != nil {
			goline.LoggerPrintln(err)
			self.handleText("Failed to download video preview.", gdk.NewColor("red"))
			return
		}
	}

	image := gtk.NewImageFromFile(previewFilePath)
	box := gtk.NewEventBox()
	box.Add(image)
	box.SetEvents(int(gdk.BUTTON_RELEASE_MASK))
	box.Connect("button-release-event", func() {
		go self.showDownloadWindow(api.LINE_OBJECT_STORAGE_URL + messageId)
	})

	label := gtk.NewLabel("Video")
	videoBox := gtk.NewEventBox()
	videoBox.Add(label)
	videoBox.SetEvents(int(gdk.BUTTON_RELEASE_MASK))
	videoBox.Connect("button-release-event", func() {
		go self.showDownloadWindow(api.LINE_OBJECT_STORAGE_URL + messageId)
	})

	table := gtk.NewTable(0, 0, false)
	table.Attach(box, 0, 1, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
	table.Attach(videoBox, 0, 1, 1, 2, gtk.FILL, gtk.FILL, 0, 0)

	self.Widget = self.tableLayout(table)
}

func (self *Sentence) showDownloadWindow(url string) {
	gdk.ThreadsEnter()
	dialog := gtk.NewFileChooserDialog("Save File",
		self.Parent.Window,
		gtk.FILE_CHOOSER_ACTION_SAVE,
		gtk.STOCK_CANCEL, gtk.RESPONSE_CANCEL,
		gtk.STOCK_SAVE, gtk.RESPONSE_ACCEPT)

	res := dialog.Run()
	if res == gtk.RESPONSE_ACCEPT {
		w := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
		filePath := dialog.GetFilename()
		w.SetTitle("Download Status")
		w.SetPosition(gtk.WIN_POS_MOUSE)
		w.SetDefaultSize(250, 150)
		label := gtk.NewLabel("Downloading...")
		w.Add(label)
		dialog.Destroy()
		w.ShowAll()
		gdk.ThreadsLeave()
		err := DownloadFile(url, filePath)
		gdk.ThreadsEnter()
		if err != nil {
			goline.LoggerPrintln(err)
			label.SetText("Download failed.")
			label.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("red"))
		} else {
			label.SetText("Download successed.")
			label.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("blue"))
		}
		gdk.ThreadsLeave()
	} else {
		dialog.Destroy()
		gdk.ThreadsLeave()
	}
}

func (self *Sentence) getNameById(id string) string {
	entity, err := goline.client.GetLineEntityById(id)
	if err != nil || entity == nil {
		return "Unknown"
	} else {
		return entity.GetName()
	}
}

func (self *Sentence) showImageWindow(id string) {
	gdk.ThreadsEnter()
	label := gtk.NewLabel("Downloading...")
	viewport := gtk.NewViewport(nil, nil)
	viewport.Add(label)
	scroll := gtk.NewScrolledWindow(nil, nil)
	scroll.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	scroll.AddWithViewPort(viewport)

	w := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	w.SetTitle("Image")
	w.SetPosition(gtk.WIN_POS_MOUSE)
	w.SetDefaultSize(500, 500)
	w.Add(scroll)
	w.ShowAll()
	gdk.ThreadsLeave()

	filePath := path.Join(goline.TempDirPath, "image", id)
	if CheckFileNotExist(filePath) {
		url := api.LINE_OBJECT_STORAGE_URL + id
		err := DownloadFile(url, filePath)
		if err != nil {
			goline.LoggerPrintln(err)
			label.SetText("Failed to download image.")
			label.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("red"))
			return
		}
	}
	gdk.ThreadsEnter()
	image := gtk.NewImageFromFile(filePath)
	viewport.Remove(label)
	viewport.Add(image)
	w.ShowAll()
	gdk.ThreadsLeave()

}

func (self *Sentence) handleImage() {
	messageId := self.Message.GetId()
	previewFilePath := path.Join(goline.TempDirPath, "preview", messageId)
	meta := self.Message.ContentMetadata
	if CheckFileNotExist(previewFilePath) {
		var previewUrl string
		if meta["PUBLIC"] == "TRUE" {
			previewUrl = meta["PREVIEW_URL"]
		} else {

			previewUrl = api.LINE_OBJECT_STORAGE_URL + messageId + "/preview"
		}
		err := DownloadFile(previewUrl, previewFilePath)
		if err != nil {
			goline.LoggerPrintln(err)
			self.handleText("Failed to download image preview.", gdk.NewColor("red"))
			return
		}
	}

	image := gtk.NewImageFromFile(previewFilePath)
	box := gtk.NewEventBox()
	box.Add(image)
	box.SetEvents(int(gdk.BUTTON_RELEASE_MASK))
	box.Connect("button-release-event", func() {
		go self.showImageWindow(messageId)
	})

	download := gtk.NewLabel("Download")
	downloadBox := gtk.NewEventBox()
	downloadBox.Add(download)
	downloadBox.SetEvents(int(gdk.BUTTON_RELEASE_MASK))
	downloadBox.Connect("button-release-event", func() {
		if meta["PUBLIC"] == "TRUE" {
			go self.showDownloadWindow(meta["DOWNLOAD_URL"])
		} else {
			go self.showDownloadWindow(api.LINE_OBJECT_STORAGE_URL + messageId)
		}
	})
	downloadTable := gtk.NewTable(0, 0, false)
	downloadTable.Attach(box, 0, 1, 0, 1, gtk.FILL, gtk.FILL, 0, 0)
	downloadTable.Attach(downloadBox, 0, 1, 1, 2, gtk.FILL, gtk.FILL, 0, 0)

	self.Widget = self.tableLayout(downloadTable)
}

func (self *Sentence) NewRandomColorFromId(id string) *gdk.Color {
	num := []byte(id)
	var (
		r uint8 = 20
		g uint8 = 20
		b uint8 = 20
	)
	if uint8(num[0])%2 == 1 {
		r += uint8(num[10]) % 100
	}
	if uint8(num[1]%2) == 1 {
		g += uint8(num[11]) % 100
	}
	if uint8(num[2]%2) == 1 {
		b += uint8(num[12]) % 100
	}

	return gdk.NewColorRGB(r, g, b)
}

func (self *Sentence) handleText(text string, color *gdk.Color) {
	fromId := self.Message.GetFrom()
	var label *gtk.Label
	if fromId == goline.client.Profile.GetMid() {
		label = gtk.NewLabel(text)
		label.SetAlignment(1, 0.5)
	} else {
		if color == nil {
			color = self.NewRandomColorFromId(fromId)
		}
		name := self.getNameById(fromId)
		label = gtk.NewLabel(name + ": " + text)
		label.SetAlignment(0, 0.5)
	}
	if color != nil {
		label.ModifyFG(gtk.STATE_NORMAL, color)
	}
	label.SetLineWrap(true)
	label.SetUseLineWrapMode(pango.WRAP_CHAR)
	label.SetSelectable(true)
	self.Widget = label
}
