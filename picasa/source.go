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
func (p *PhotoSource) Loop() {
	defer func() {
		p.done <- true
	}()
	token, err := p.config.Token()
	if err != nil {
		p.config.context.Errorf("Error retrieving token: %s", err)
		return
	}
        p.config.context.Infof("Token: %#v\n", token)
	photos, err := findPhotos(token, urlfetch.Client(p.config.context))
	if err != nil {
		p.config.context.Errorf("Error finding photos: %s", err)
		return
	}
	for i, _ := range photos {
                p.config.context.Infof("Photo: %#v", &photos[i])
		p.sink <- &photos[i]
	}
}
