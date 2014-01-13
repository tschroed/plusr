package picasa

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"code.google.com/p/goauth2/oauth"
)

const (
	// TODO(trevors): Support pagination.
	GDATA_FEED    = "https://picasaweb.google.com/data/feed/api/user/default?kind=photo&max-results=1000&q=plusr&imgmax=d"
	GDATA_VERSION = "2"
)

type Content struct {
	Src  string `xml:"src,attr"`
	Type string `xml:"type,attr"`
}

type Photo struct {
	Contents   Content `xml:"content"`
	Id         string  `xml:"id"`
	Version    string  `xml:"version"`
	Title      string  `xml:"title"`
	Album      string  `xml:"albumtitle"`
	GlobalUid  string
	httpClient *http.Client
	token      *oauth.Token
}

func (p *Photo) Body() (io.ReadCloser, error) {
        resp, err := get(p.Contents.Src, p.token, p.httpClient)
        if err != nil {
                return nil, err
        }
        return resp.Body, nil
}

func (p *Photo) Metadata() (globalUid, album, title string) {
	return p.GlobalUid, p.Album, p.Title
}

type PhotoFeed struct {
	XMLName xml.Name `xml:"feed"`
	Photos  []Photo  `xml:"entry"`
}

func parseFeed(text []byte) PhotoFeed {
	feed := PhotoFeed{}
	xml.Unmarshal([]byte(text), &feed)
	for i, _ := range feed.Photos {
		p := &feed.Photos[i]
		p.GlobalUid = fmt.Sprintf("picasa:%s", p.Id)
	}
	return feed
}

func authHeader(token *oauth.Token) (string, string) {
	return "Authorization", fmt.Sprintf("Bearer %s", token.AccessToken)
}

func gdataHeader() (string, string) {
	return "Gdata-version", GDATA_VERSION
}

func get(path string, token *oauth.Token, client *http.Client) (*http.Response, error) {
	var k, v string
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	k, v = gdataHeader()
	req.Header.Add(k, v)
	k, v = authHeader(token)
	req.Header.Add(k, v)
	return client.Do(req)
}

func getGdataFeed(token *oauth.Token, client *http.Client) ([]byte, error) {
	resp, err := get(GDATA_FEED, token, client)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

func findPhotos(token *oauth.Token, client *http.Client) ([]Photo, error) {
	text, err := getGdataFeed(token, client)
	if err != nil {
		return nil, err
	}
	feed := parseFeed(text)
	for i, _ := range feed.Photos {
		p := &feed.Photos[i]
		p.httpClient = client
		p.token = token
	}
	return feed.Photos, nil
}
