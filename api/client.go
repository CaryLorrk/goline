package api

import (
	"net/http"
	"sort"
	"sync"

	"git.apache.org/thrift.git/lib/go/thrift"
	prot "github.com/carylorrk/goline/protocol"
)

const (
	LINE_DOMAIN = "http://gd2.line.naver.jp"

	LINE_HTTP_URL           = LINE_DOMAIN + "/api/v4/TalkService.do"
	LINE_HTTP_IN_URL        = LINE_DOMAIN + "/P4"
	LINE_CERTIFICATE_URL    = LINE_DOMAIN + "/Q"
	LINE_SESSION_LINE_URL   = LINE_DOMAIN + "/authct/v1/keys/line"
	LINE_SESSION_NAVER_URL  = LINE_DOMAIN + "/authct/v1/keys/naver"
	LINE_OBJECT_STORAGE_URL = "http://os.line.naver.jp/os/m/"
	LINE_STICKER_URL        = "http://dl.stickershop.line.naver.jp/products/0/0/"
	LINE_USER_AGENT         = "DESKTOP:MAC:10.9.4-MAVERICKS-x64(3.7.0)"
	LINE_X_LINE_APPLICATION = "DESKTOPMAC\t3.7.0\tMAC\t10.9.4-MAVERICKS-x64"
)

type ContactSlice []*prot.Contact

func (s ContactSlice) Less(i, j int) bool {
	return s[i].GetDisplayName() < s[j].GetDisplayName()
}

func (s ContactSlice) Swap(i, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}

func (s ContactSlice) Len() int {
	return len(s)
}

type GroupSlice []*prot.Group

func (s GroupSlice) Less(i, j int) bool {
	return s[i].GetName() < s[j].GetName()
}

func (s GroupSlice) Swap(i, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}

func (s GroupSlice) Len() int {
	return len(s)
}

type LineClient struct {
	Profile   *prot.Profile
	Provider  prot.IdentityProvider
	Contacts  ContactSlice
	Groups    GroupSlice
	Rooms     []*prot.Room
	AuthToken string
	IP        string
	Hostname  string
	client    *prot.TalkServiceClient
	header    *http.Header
	revision  int64
	lock      sync.Mutex
}

func NewLineClient() (*LineClient, error) {
	transport, err := thrift.NewTHttpPostClient(LINE_HTTP_URL)
	if err != nil {
		return nil, err
	}

	httpTrans := transport.(*thrift.THttpClient)
	header := &http.Header{}
	header.Add("User-Agent", LINE_USER_AGENT)
	httpTrans.SetHeader("User-Agent", LINE_USER_AGENT)
	header.Add("X-LINE-Application", LINE_X_LINE_APPLICATION)
	httpTrans.SetHeader("X-Line-Application", LINE_X_LINE_APPLICATION)
	protocol := thrift.NewTCompactProtocol(transport)
	client := prot.NewTalkServiceClientProtocol(transport, protocol, protocol)
	return &LineClient{client: client,
		IP: lookupIP(), Hostname: lookupHostname(),
		header: header}, nil
}

func (self *LineClient) RefreshRevision() (int64, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	var err error
	self.revision, err = self.client.GetLastOpRevision()
	if err != nil {
		self.revision = 0
		return 0, err
	}
	return self.revision, nil
}

func (self *LineClient) RefreshProfile() (*prot.Profile, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	var err error
	self.Profile, err = self.client.GetProfile()
	return self.Profile, err
}

func (self *LineClient) RefreshContacts() ([]*prot.Contact, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	ids, err := self.client.GetAllContactIds()
	if err != nil {
		return nil, err
	}

	self.Contacts, err = self.client.GetContacts(ids)
	sort.Sort(self.Contacts)
	return self.Contacts, err
}

func (self *LineClient) RefreshGroups() ([]*prot.Group, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	joinedIds, err := self.client.GetGroupIdsJoined()
	if err != nil {
		return nil, err
	}

	invitedIds, err := self.client.GetGroupIdsInvited()
	if err != nil {
		return nil, err
	}

	joinedGroups, err := self.client.GetGroups(joinedIds)
	if err != nil {
		return nil, err
	}

	self.Groups, err = self.client.GetGroups(invitedIds)
	if err != nil {
		return nil, err
	}

	self.Groups = append(self.Groups, joinedGroups...)
	sort.Sort(self.Groups)

	return self.Groups, err

}

func (self *LineClient) RefreshRooms() ([]*prot.Room, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	start := int32(1)
	count := int32(50)
	self.Rooms = make([]*prot.Room, 0)
	for {
		channel, err := self.client.GetMessageBoxWrapUpList(start, count)
		if err != nil {
			return nil, err
		}
		for _, messageBoxWrapUp := range channel.MessageBoxWrapUpList {
			messageBox := messageBoxWrapUp.MessageBox
			if messageBox.MidType == prot.MIDType_ROOM {
				rooms, err := self.client.GetRoom(messageBox.Id)
				if err != nil {
					return nil, err
				}
				self.Rooms = append(self.Rooms, rooms)
			}
		}
		if len(channel.MessageBoxWrapUpList) == int(count) {
			start += count
		} else {
			break
		}
	}
	return self.Rooms, nil
}

func (self *LineClient) GetContactById(id string) *prot.Contact {
	for _, contact := range self.Contacts {
		if contact.GetMid() == id {
			return contact
		}
	}

	return nil
}

func (self *LineClient) GetGroupById(id string) *prot.Group {
	for _, group := range self.Groups {
		if group.GetId() == id {
			return group
		}
	}
	return nil
}

func (self *LineClient) GetRoomById(id string) *prot.Room {
	for _, room := range self.Rooms {
		if room.GetMid() == id {
			return room
		}
	}
	return nil
}

func (self *LineClient) GetLineEntityById(id string) (LineEntity, error) {
	contact := self.GetContactById(id)
	if contact != nil {
		return NewLineContactWrapper(contact), nil
	}
	group := self.GetGroupById(id)
	if group != nil {
		return NewLineGroupWrapper(group), nil
	}
	room := self.GetRoomById(id)
	if room != nil {
		return NewLineRoomWrapper(room), nil
	}

	_, err := self.RefreshContacts()
	if err != nil {
		return nil, err
	}
	contact = self.GetContactById(id)
	if contact != nil {
		return NewLineContactWrapper(contact), nil
	}

	_, err = self.RefreshGroups()
	if err != nil {
		return nil, err
	}
	group = self.GetGroupById(id)
	if group != nil {
		return NewLineGroupWrapper(group), nil
	}

	_, err = self.RefreshRooms()
	if err != nil {
		return nil, err
	}
	room = self.GetRoomById(id)
	if room != nil {
		return NewLineRoomWrapper(room), nil
	}
	return nil, nil
}

func (self *LineClient) GetHeader() *http.Header {
	return self.header
}
