package api

import (
	prot "github.com/carylorrk/goline/protocol"
)

func (self *LineClient) GetMessageBox(id string) (*prot.TMessageBox, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	messageWrapUp, err := self.client.GetMessageBoxCompactWrapUp(id)
	if err != nil {
		return nil, err
	}
	return messageWrapUp.GetMessageBox(), nil
}

func (self *LineClient) GetRecentMessages(messageBox *prot.TMessageBox, count int32) ([]*prot.Message, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.client.GetRecentMessages(messageBox.GetId(), count)
}

func (self *LineClient) SendText(id string, text string) (*prot.Message, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	message := &prot.Message{To: id, Text: text}
	return self.client.SendMessage(0, message)
}

func (self *LineClient) FetchNewOperations(count int32) ([]*prot.Operation, error) {
	self.lock.Lock()
	operations, err := self.client.FetchOperations(self.revision, count)
	self.lock.Unlock()
	if err != nil {
		return operations, err
	}
	for _, operation := range operations {
		if operation.GetRevision() > self.revision {
			self.revision = operation.GetRevision()
		}
	}
	return operations, err
}
