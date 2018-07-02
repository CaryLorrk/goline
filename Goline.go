package main

import (
	"encoding/json"
	"github.com/carylorrk/goline/api"
	"io"
	"log"
	"os"
	"os/user"
	"path"
	"runtime"
)

type Goline struct {
	Id          string          `json:"Id"`
	Password    string          `json:"Password"`
	AuthToken   string          `json:"AuthToken"`
	Remember    bool            `json:"Remember"`
	DataDirPath string          `json:"-"`
	TempDirPath string          `json:"-"`
	client      *api.LineClient `json:"-"`
	logger      *log.Logger     `json:"-"`
}

func NewGoline() (goline *Goline, err error) {
	goline = &Goline{}
	err = goline.setupDirPath()
	if err != nil {
		return
	}

	err = goline.setupLogger()
	if err != nil {
		return
	}

	err = goline.setupSettings()
	if err != nil {
		goline.LoggerPrintln(err)
		return
	}
	return
}

func (self *Goline) setupDirPath() (err error) {
	var usr *user.User
	usr, err = user.Current()
	if err != nil {
		return
	}
	self.DataDirPath = path.Join(usr.HomeDir, ".goline")
	err = os.MkdirAll(self.DataDirPath, os.FileMode(0700))
	if err != nil {
		return
	}

	self.TempDirPath = path.Join(os.TempDir(), "goline")
	err = os.MkdirAll(self.TempDirPath, os.FileMode(0700))
	if err != nil {
		return
	}

	previewPath := path.Join(self.TempDirPath, "preview")
	err = os.MkdirAll(previewPath, os.FileMode(0700))
	if err != nil {
		return
	}

	stickerPath := path.Join(self.TempDirPath, "sticker")
	err = os.MkdirAll(stickerPath, os.FileMode(0700))
	if err != nil {
		return
	}

	thumbnailPath := path.Join(self.TempDirPath, "thumbnail")
	err = os.MkdirAll(thumbnailPath, os.FileMode(0700))
	if err != nil {
		return
	}

	imagePath := path.Join(self.TempDirPath, "image")
	err = os.MkdirAll(imagePath, os.FileMode(0700))
	return
}

func (self *Goline) loadConfigFile() (*os.File, error) {
	configFilePath := path.Join(self.DataDirPath, "settings.json")
	configFile, err := os.OpenFile(configFilePath, os.O_RDWR|os.O_CREATE, os.FileMode(0666))
	if err != nil {
		self.LoggerPrintln(err)
		return nil, err
	}
	return configFile, err
}

func (self *Goline) SaveSettings() error {
	configFile, err := self.loadConfigFile()
	if err != nil {
		self.LoggerPrintln(err)
		return err
	}
	defer configFile.Close()
	jsonEncoder := json.NewEncoder(configFile)
	err = jsonEncoder.Encode(self)
	return err
}

func (self *Goline) setupSettings() error {
	configFile, err := self.loadConfigFile()
	if err != nil {
		self.LoggerPrintln(err)
		return err
	}
	jsonDecoder := json.NewDecoder(configFile)
	err = jsonDecoder.Decode(self)
	if err != nil {
		err = configFile.Truncate(0)
		if err != nil {
			self.LoggerPrintln(err)
			return err
		}
		self.logger.Println("Create new setting file.")
		jsonEncoder := json.NewEncoder(configFile)
		err = jsonEncoder.Encode(self)
		if err != nil {
			self.LoggerPrintln(err)
			return err
		}
	}
	configFile.Close()
	return err
}

func (self *Goline) setupLogger() error {
	logFilePath := path.Join(self.TempDirPath, "log")
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	writer := io.MultiWriter(logFile, os.Stderr)
	self.logger = log.New(writer, "Goline: ", log.LstdFlags)
	return nil
}

func (self *Goline) LoggerPrintln(v ...interface{}) {
	pc, file, line, ok := runtime.Caller(1)
	if ok {
		self.logger.Println(pc, file, line, v)
	} else {
		self.logger.Panicln(v)
	}
}
