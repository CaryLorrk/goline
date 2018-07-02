package main

import (
	"io"
	"net/http"
	"os"

	"github.com/mattn/go-gtk/gtk"
)

func RunAlertMessage(parent *gtk.Window, format string) {
	dialog := gtk.NewMessageDialog(parent, 0, gtk.MESSAGE_WARNING, gtk.BUTTONS_OK, format)
	dialog.Run()
	dialog.Destroy()
}

func RunErrorMessage(parent *gtk.Window, format string) {
	dialog := gtk.NewMessageDialog(parent, 0, gtk.MESSAGE_WARNING, gtk.BUTTONS_OK, format)
	dialog.Run()
	dialog.Destroy()
}

func CheckFileNotExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return true
	}
	return false
}

func DownloadFile(url, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		goline.LoggerPrintln(err)
		return err
	}
	defer file.Close()
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		goline.LoggerPrintln(err)
		return err
	}
	req.Header = *goline.client.GetHeader()
	res, err := client.Do(req)
	if err != nil {
		goline.LoggerPrintln(err)
		return err
	}
	defer res.Body.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		goline.LoggerPrintln(err)
	}
	return err
}
