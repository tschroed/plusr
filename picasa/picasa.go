package picasa

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"

	"appengine"
	"appengine/urlfetch"

	"code.google.com/p/goauth2/oauth"
)

const (
	// TODO(trevors): Support pagination.
	GDATA_FEED    = "https://picasaweb.google.com/data/feed/api/user/default?kind=photo&max-results=1000&q=plusr&imgmax=d"
	GDATA_VERSION = "2"
)

type Content struct {
	Src string `xml:"src,attr"`
}

type Photo struct {
	Contents Content `xml:"content"`
	Id       string  `xml:"id"`
	Version  string  `xml:"version"`
	Title    string  `xml:"title"`
	Album    string  `xml:"albumtitle"`
}

type PhotoFeed struct {
	XMLName xml.Name `xml:"feed"`
	Photos  []Photo  `xml:"entry"`
}

func parseFeed(text []byte) PhotoFeed {
	feed := PhotoFeed{}
	xml.Unmarshal([]byte(text), &feed)
	return feed
}

func authHeader(token *oauth.Token) (string, string) {
	return "Authorization", fmt.Sprintf("Bearer %s", token.AccessToken)
}

func gdataHeader() (string, string) {
	return "Gdata-version", GDATA_VERSION
}

func getGdataFeed(token *oauth.Token, client *http.Client) ([]byte, error) {
	var k, v string
	req, err := http.NewRequest("GET", GDATA_FEED, nil)
	if err != nil {
		return nil, err
	}
	k, v = gdataHeader()
	req.Header.Add(k, v)
	k, v = authHeader(token)
	req.Header.Add(k, v)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

func findPhotos(token *oauth.Token, client *http.Client) ([]Photo, error) {
	text, err := getGdataFeed(token, client)
	if err != nil {
		return nil, err
	}
	feed := parseFeed(text)
	return feed.Photos, nil
}

func PhotoFeedHandler(w http.ResponseWriter, r *http.Request) {
	uc := userConfig{context: appengine.NewContext(r)}
	token, err := uc.Token()
	if err != nil {
		uc.context.Errorf("Token(): %s", err)
		return
	}
	photos, err := findPhotos(token, urlfetch.Client(uc.context))
	if err != nil {
		uc.context.Errorf("findPhotos(): %s", err)
		return
	}
	for _, p := range photos {
		uc.context.Infof("Photo: %s (%s)", p.Title, p.Contents.Src)
	}
}
