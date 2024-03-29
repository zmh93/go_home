package meander

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

var APIKey = os.Getenv("GOOGLE_PLACE_API")

type Place struct {
	*googleGeometry `json:"geometry"`
	Name            string         `json:"name"`
	Icon            string         `json:"icon"`
	Photos          []*googlePhoto `json:"photos"`
	Vicinity        string         `json:"vicinity"`
}

func (p *Place) Public() interface{} {
	return map[string]interface{}{
		"name":     p.Name,
		"icon":     p.Icon,
		"photos":   p.Photos,
		"vicinity": p.Vicinity,
		"lat":      p.Lat,
		"lng":      p.Lng,
	}
}

type Query struct {
	Lat          float64
	Lng          float64
	Journey      []string
	Radius       int
	CostRangeStr string
}

func (q *Query) find(types string) (*googleResponse, error) {
	u := "https://maps.googleapis.com/maps/api/place/nearbysearch/json"
	vals := make(url.Values)
	vals.Set("location", fmt.Sprintf("%g,%g", q.Lat, q.Lng))
	vals.Set("radius", fmt.Sprintf("%d", q.Radius))
	vals.Set("type", types)
	vals.Set("key", APIKey)
	if len(q.CostRangeStr) > 0 {
		r, err := ParseCostRange(q.CostRangeStr)
		if err != nil {
			return nil, err
		}
		vals.Set("minprice", fmt.Sprintf("%d", int(r.From)-1))
		vals.Set("maxprice", fmt.Sprintf("%d", int(r.To)-1))
	}
	googleUrl := u + "?" + vals.Encode()
	fmt.Println("googleUrl", googleUrl)
	res, err := http.Get(googleUrl)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var response googleResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

//Run runs the query concurrently,and return the results
func (q *Query) Run() []interface{} {
	rand.Seed(time.Now().UnixNano())
	var w sync.WaitGroup
	var l sync.Mutex
	places := make([]interface{}, len(q.Journey))
	for i, r := range q.Journey {
		w.Add(1)
		go func(types string, i int) {
			defer w.Done()
			response, err := q.find(types)
			if err != nil {
				log.Println("failed to find places:", err)
				return
			}
			if len(response.Results) == 0 {
				log.Println("no places found for", types)
				return
			}
			for _, result := range response.Results {
				for _, photo := range result.Photos {
					photo.URL = "http://maps.googleapis.com/maps/api/place/photo?" +
						"maxwidth=1000&photoreference=" + photo.PhotoRef + "&key=" + APIKey
				}
			}
			randI := rand.Intn(len(response.Results))
			l.Lock()
			places[i] = response.Results[randI]
			l.Unlock()
		}(r, i)
	}
	w.Wait()
	return places
}

type googleResponse struct {
	Results []Place `json:"results"`
}

type googleGeometry struct {
	googleLocation `json:"location"`
}

type googleLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type googlePhoto struct {
	Height   int    `json:"height"`
	Width    int    `json:"width"`
	PhotoRef string `json:"photo_ref"`
	URL      string `json:"url"`
}
