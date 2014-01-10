package flickr

import (
	"appengine/datastore"

	"code.google.com/p/flickgo"
)

func (uc *userConfig) photoKey(globalUID string) *datastore.Key {
	return datastore.NewKey(uc.context, "flickgo.TicketStatus",
		globalUID, 0, uc.rootUserKey())
}

func (uc *userConfig) maybeLoadTicketStatus(globalUID string) (*flickgo.TicketStatus, error) {
	status := &flickgo.TicketStatus{}
	key := uc.photoKey(globalUID)
	if err := datastore.Get(uc.context, key, status); err != nil {
		return nil, err
	}
	return status, nil
}

func (uc *userConfig) saveTicketStatus(globalUID string, status *flickgo.TicketStatus) error {
	key := uc.photoKey(globalUID)
	if _, err := datastore.Put(uc.context, key, status); err != nil {
		return err
	}
	return nil
}
