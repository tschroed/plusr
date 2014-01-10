package flickr

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"appengine"
	"appengine/user"

	"code.google.com/p/flickgo"
)

func (uc *userConfig) Do(req *http.Request) (resp *http.Response, err error) {
	consumer := uc.oauthConsumer()
	tok, _ := uc.accessToken()
	return consumer.Do(req, tok)
}

func (uc *userConfig) Head(url string) (resp *http.Response, err error) {
	return nil, errors.New("Not Implemented.")
}

func (uc *userConfig) Get(url string) (resp *http.Response, err error) {
	consumer := uc.oauthConsumer()
	tok, _ := uc.accessToken()
	return consumer.Get(url, nil, tok)
}

func (uc *userConfig) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	return nil, errors.New("Not Implemented.")
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

func readImage() ([]byte, error) {
	f, err := os.Open("/tmp/test-pattern.jpg")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

func UploadGarbage(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	uc := userConfig{context: c, rootUser: user.Current(c).String()}
	fc := flickgo.New(APIKey, APISecret, &uc)
//	fc.Logger = uc.context
        fc.DisableAuth = true
	c.Infof("==== TESTING LOGIN USING LEGACY CODE ====")
	username, id, err := fc.TestLogin()
	c.Infof("username, id, err: %v, %v, %v", id, username, err)
        consumer := uc.oauthConsumer()
        c.Infof("==== TESTING LOGIN USING NEW CODE ====")
        host := "www.flickr.com"
        path := "/services/rest?method=flickr.test.login"
        req, err := http.NewRequest("GET",
                                    fmt.Sprintf("http://%s%s", host, path), nil)
        if err != nil {
                uc.context.Errorf("NewRequest: %v", err)
                return
        }
        tok, _ := uc.accessToken()
        resp, err := consumer.Do(req, tok)
        if err != nil {
                uc.context.Errorf("Do error: %v", err)
                return
        }
        b, err := ioutil.ReadAll(resp.Body)
        defer resp.Body.Close()
        if err != nil {
                uc.context.Errorf("ReadAll error: %v", err)
                return
        }
        uc.context.Infof("%v", bytes.NewBuffer(b).String())
	c.Infof("==== TESTING UPLOAD USING NEW CODE ====")
	img, err := readImage()
	if err == nil {
		buf := bytes.NewBuffer(img)
		tick, err := fc.Upload(fmt.Sprintf("Test Pattern at %v", time.Now()),
			buf.Bytes(), nil)
		uc.context.Infof("tick, err: %v, %v", tick, err)
        OUT:
                for {
                        statuses, err := fc.CheckTickets([]string{tick})
                        switch {
                        case err != nil:
                                uc.context.Errorf("CheckTickets: %v", err)
                                break OUT
                        case statuses[0].Invalid != "":
                                uc.context.Errorf("Invalid: %v", statuses[0].Invalid)
                                break OUT
                        case statuses[0].Complete == "2":
                                uc.context.Errorf("Conversion failed.")
                                break OUT
                        case statuses[0].Complete == "1":
                                uc.context.Infof("All done!")
                                break OUT
                        }
                        time.Sleep(2 * time.Second)
                }
	}
	/*
	        c.Infof("==== PLAIN OLD POST VIA CONTEXT ====")
		buf = bytes.NewBuffer("This is a bunch of crap. Foo.\r\n")
	        client := urlfetch.Client(uc.context)
	        resp, err = client.Post("http://www.google.com/", "text/plain", buf)
	        uc.context.Infof("Post: %#v, %#v", resp, err)
	*/
}
