package picasa

import (
	"appengine/urlfetch"
)

type PhotoSource struct {
	config *userConfig
	done   chan bool
	sink   chan *Photo
}

func NewPhotoSource(config *userConfig, sink chan *Photo, done chan bool) *PhotoSource {
	return &PhotoSource{config: config, done: done, sink: sink}
}

// Blocking
func (p *PhotoSource) Start() {
	defer func() {
		p.done <- true
	}()
	token, err := p.config.Token()
	if err != nil {
		p.config.context.Errorf("Error retrieving token: %s", err)
		return
	}
	photos, err := findPhotos(token, urlfetch.Client(p.config.context))
	if err != nil {
		p.config.context.Errorf("Error finding photos: %s", err)
		return
	}
	for _, photo := range photos {
		p.sink <- &photo
	}
}
