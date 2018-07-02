package main

import (
	"os"
	"runtime"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
)

var goline *Goline

func gtkInit() {
	runtime.GOMAXPROCS(10)
	glib.ThreadInit(nil)
	gdk.ThreadsInit()
	gdk.ThreadsEnter()
	gtk.Init(&os.Args)
}

func main() {
	gtkInit()
	var err error
	goline, err = NewGoline()
	if err != nil {
		panic(err)
	}
	goline.LoggerPrintln("Start Goline.")
	loginWindow := NewLoginWindow()
	loginWindow.Window.ShowAll()
	loginWindow.CheckAuthToken()
	gtk.Main()
}
