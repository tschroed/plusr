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
        Body() io.ReadCloser
}

type PhotoSink struct {
	config *userConfig
	done   chan bool
	source   chan Photo
}

func NewPhotoSink(config *userConfig, source chan Photo, done chan bool) *PhotoSink {
	return &PhotoSink{config: config, done: done, source: source}
}

// Blocking
func (p *PhotoSink) Loop() {
        for {
                select {
                case ph := <-p.source:
                        _, _, title := ph.Metadata()
                        p.config.context.Infof("About to upload %v", title)
                        p.uploadPhoto(ph)
                case <-p.done:
                        return
                }
        }
}

func (p *PhotoSink) uploadPhoto(photo Photo) {
	fc := flickgo.New(APIKey, APISecret, p.config)
	fc.Logger = p.config.context
	p.config.context.Infof("==== TESTING UPLOAD USING NEW CODE ====")
        // guid, album, title := photo.Metadata()
        _, _, title := photo.Metadata()
        defer photo.Body().Close()
	img, err := ioutil.ReadAll(photo.Body())
	if err == nil {
		buf := bytes.NewBuffer(img)
		tick, err := fc.Upload(title, buf.Bytes(), nil)
		p.config.context.Infof("tick, err: %v, %v", tick, err)
        OUT:
                for {
                        statuses, err := fc.CheckTickets([]string{tick})
                        switch {
                        case err != nil:
                                p.config.context.Errorf("CheckTickets: %v", err)
                                break OUT
                        case statuses[0].Invalid != "":
                                p.config.context.Errorf("Invalid: %v", statuses[0].Invalid)
                                break OUT
                        case statuses[0].Complete == "2":
                                p.config.context.Errorf("Conversion failed.")
                                break OUT
                        case statuses[0].Complete == "1":
                                p.config.context.Infof("All done!")
                                break OUT
                        }
                        time.Sleep(2 * time.Second)
                }
        }
}
