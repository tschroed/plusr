package flickr

import (
	"bytes"
	"io"
	"io/ioutil"
	"time"

	"code.google.com/p/flickgo"
)

type Photo interface {
	Metadata() (globalUid, album, title string)
	Body() (io.ReadCloser, error)
}

type PhotoSink struct {
	config *userConfig
	done   chan bool
	source chan Photo
}

func NewPhotoSink(config *userConfig, source chan Photo, done chan bool) *PhotoSink {
	return &PhotoSink{config: config, done: done, source: source}
}

// Blocking
func (p *PhotoSink) Loop() {
	for {
		select {
		case ph := <-p.source:
			guid, _, title := ph.Metadata()
			p.config.context.Infof("About to upload %v", title)
			tick, err := p.config.maybeLoadTicketStatus(guid)
			if err != nil {
				p.config.context.Warningf("maybeLoadTicketStatus: %#v", err)
			}
			if tick == nil {
				tick = p.uploadPhoto(ph)
			}
			if tick == nil {
				p.config.context.Warningf("Didn't get a ticket")
				return
			}
			p.waitTicket(ph, tick)
		case <-p.done:
			return
		}
	}
}

func (p *PhotoSink) waitTicket(photo Photo, tick *flickgo.TicketStatus) {
	guid, _, _ := photo.Metadata()
	for {
		fc := flickgo.New(APIKey, APISecret, p.config)
		statuses, err := fc.CheckTickets([]string{tick.ID})
		if statuses != nil && len(statuses) > 0 {
			p.config.saveTicketStatus(guid, &statuses[0])
		}
		switch {
		case err != nil:
			p.config.context.Errorf("CheckTickets: %v", err)
			return
		case statuses[0].Invalid != "":
			p.config.context.Errorf("Invalid: %v", statuses[0].Invalid)
			return
		case statuses[0].Complete == "2":
			p.config.context.Errorf("Conversion failed.")
			return
		case statuses[0].Complete == "1":
			p.config.context.Infof("All done!")
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func (p *PhotoSink) uploadPhoto(photo Photo) *flickgo.TicketStatus {
	fc := flickgo.New(APIKey, APISecret, p.config)
	fc.Logger = p.config.context
	p.config.context.Infof("==== TESTING UPLOAD USING NEW CODE ====")
	// guid, album, title := photo.Metadata()
	_, _, title := photo.Metadata()
	body, err := photo.Body()
	if err != nil {
		p.config.context.Errorf("Error getting photo body: %#v", err)
		return nil
	}
	defer body.Close()
	img, err := ioutil.ReadAll(body)
	if err == nil {
		buf := bytes.NewBuffer(img)
		tick, err := fc.Upload(title, buf.Bytes(), nil)
		p.config.context.Infof("tick, err: %v, %v", tick, err)
		status, err := fc.CheckTickets([]string{tick})
		p.config.context.Infof("CheckTickets error: %v", err)
		return &status[0]
	} else {
		p.config.context.Errorf("ReadAll: %v", err)
	}
	return nil
}
