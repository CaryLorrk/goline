package api

import (
	prot "github.com/carylorrk/goline/protocol"
	"strconv"
)

type LineEntity interface {
	GetId() string
	GetName() string
	Refresh(*LineClient)
}

type LineContactWrapper struct {
	contact *prot.Contact
}

func NewLineContactWrapper(contact *prot.Contact) *LineContactWrapper {
	return &LineContactWrapper{contact}
}

func (self *LineContactWrapper) GetId() string {
	return self.contact.GetMid()
}

func (self *LineContactWrapper) GetName() string {
	return self.contact.DisplayName
}

func (self *LineContactWrapper) Refresh(client *LineClient) {
	client.RefreshContacts()
}

func (self *LineContactWrapper) GetContact() *prot.Contact {
	return self.contact
}

type LineGroupWrapper struct {
	group *prot.Group
}

func NewLineGroupWrapper(group *prot.Group) *LineGroupWrapper {
	return &LineGroupWrapper{group}
}

func (self *LineGroupWrapper) GetId() string {
	return self.group.GetId()
}

func (self *LineGroupWrapper) GetName() string {
	return self.group.GetName()
}

func (self *LineGroupWrapper) Refresh(client *LineClient) {
	client.RefreshGroups()
}

type LineRoomWrapper struct {
	room *prot.Room
	name string
}

func NewLineRoomWrapper(room *prot.Room) *LineRoomWrapper {
	return &LineRoomWrapper{room: room}
}

func (self *LineRoomWrapper) GetId() string {
	return self.room.GetMid()
}

func (self *LineRoomWrapper) GetName() string {
	if self.name == "" {
		contacts := self.room.GetContacts()
		for idx, contact := range contacts {
			if idx >= 3 {
				self.name += "..."
				break
			}
			self.name += contact.GetDisplayName()
		}
		self.name += "(" + strconv.Itoa(len(contacts)) + ")"
	}
	return self.name
}

func (self *LineRoomWrapper) Refresh(client *LineClient) {
	client.RefreshRooms()
}
