package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/itchyny/gojq"
)

func main() {

	latPtr := flag.Float64("lat", 0, "Latitude")
	lonPtr := flag.Float64("lon", 0, "Longitude")
	maxPtr := flag.Int("max", 10, "Maximum number of results")
	distPtr := flag.Float64("dist", 5, "Maximum distance away to consider")

	flag.Parse()

	lat := *latPtr
	lon := *lonPtr
	max := *maxPtr
	dist := *distPtr

	wide := 0.04
	url := fmt.Sprintf("https://petrolspy.com.au/webservice-1/station/box?neLat=%0.14f&neLng=%0.14f&swLat=%0.14f&swLng=%0.14f", lat-wide, lon-wide, lat+wide, lon+wide)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error getting weather data:", err)
		os.Exit(1)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		os.Exit(1)
	}

	//fmt.Printf("%s\n", body)

	var data map[string]any

	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		os.Exit(1)
	}

	query := fmt.Sprintf(`
		def latitude: %0.14f;
		def longitude: %0.14f;
		def rows: %d;
		def max_dist: %0.14f;
		def to_radians($degrees): ($degrees | tonumber) * (3.14159265359 / 180);
		def round(precision):.*pow(10;precision)|round/pow(10;precision);
		def distance(lat1;lon1;lat2;lon2):
			pow(to_radians(lon2)-to_radians(lon1);2)+pow(to_radians(lat2)-to_radians(lat1);2)|sqrt|. * 5450;
	[.message.list[]|
		select(.name | contains("Costco") | not)|
		select(.updated < 24)|
		[
			select(.prices.U91.amount != null)|
			select(((now-((.prices.U91.updated//now)/1000))/3600|floor) < 24)|
			{
				"type": "U91",
				"price":.prices.U91.amount,
				"updated":((now-((.prices.U91.updated//now)/1000))/3600|floor),
				"name":.name,
				"distance":distance(.location.y;.location.x;latitude;longitude)|round(2)
			}|select(.distance < max_dist)
		]+[
			select(.prices.E10.amount != null)|
			select(((now-((.prices.E10.updated//now)/1000))/3600|floor) < 24)|
			{
				"type": "E10",
				"price":.prices.E10.amount,
				"updated":((now-((.prices.E10.updated//now)/1000))/3600|floor),
				"name":.name,
				"distance":distance(.location.y;.location.x;latitude;longitude)|round(2)
			}|select(.distance < max_dist)	
		]|.[]
	]|
	[
		sort_by(.price)|.[0:rows]|.[]|"\(.type)@\(.price):\(.name) \(.distance)km"
	]|join("\n")`, lat, lon, max, dist)

	// q, _ := gojq.Parse(query)

	// fmt.Printf("http %s | jq -r '%s'\n\n", url, q.String())
	fmt.Print(jq(query, data))
}

func jq(search string, data any) string {
	query, _ := gojq.Parse(search)
	//fmt.Println(query.String())
	iter := query.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}

		if s, ok := v.(string); ok {
			return s
		}

		if err, ok := v.(error); ok {
			log.Fatal(err.Error())
		}
		//lint:ignore SA4004 easiest way to get result as string
		return fmt.Sprint(v)
	}
	return ""
}
