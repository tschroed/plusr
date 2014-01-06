package flickr

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"appengine"
	"appengine/user"

	"code.google.com/p/flickgo"
)

func (uc *userConfig) Get(url string) (resp *http.Response, err error) {
	consumer := uc.oauthConsumer()
	tok, _ := uc.accessToken()
	return consumer.Get(url, nil, tok)
}

func (uc *userConfig) Post(url string, bodyType string, body io.Reader) (resp *http.Response, err error) {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(b)
	consumer := uc.oauthConsumer()
	tok, _ := uc.accessToken()
	return consumer.Post(url, buf.String(), nil, tok)
}

func UploadGarbage(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	uc := userConfig{context: c, rootUser: user.Current(c).String()}
	uc.oauthConsumer().Debug(true)
	fc := flickgo.New(APIKey, APISecret, &uc)
	username, _, err := fc.TestLogin()
	c.Infof("username, err: %v, %v", username, err)
	buf := bytes.NewBufferString("This is a bunch of crap.\r\n")
	tick, err := fc.Upload("Garbage", buf.Bytes(), nil)
	uc.context.Infof("tick, err: %v, %v", tick, err)
}
