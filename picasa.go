package picasa

import (
  "encoding/xml"
)

type Content struct {
  Src string `xml:"src,attr"`
}

type Photo struct {
  Contents Content `xml:"content"`
  Id string `xml:"id"`
  Version string `xml:"version"`
  Title string `xml:"title"`
  Album string `xml:"albumtitle"`
}

type PhotoFeed struct {
  XMLName xml.Name `xml:"feed"`
  Photos []Photo `xml:"entry"`
}

func ParseFeed(text []byte) PhotoFeed {
  feed := PhotoFeed{}
  xml.Unmarshal([]byte(text), &feed)
  return feed
}
