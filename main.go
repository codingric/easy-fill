package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/itchyny/gojq"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run main.go <latitude> <longitude>")
		os.Exit(1)
	}

	lat, err := strconv.ParseFloat(os.Args[1], 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	lon, err := strconv.ParseFloat(os.Args[2], 64)
	if err != nil {
		fmt.Println(err)
		return
	}

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

	var data map[string]any

	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		os.Exit(1)
	}

	query := fmt.Sprintf(`
		def to_radians($degrees): ($degrees | tonumber) * (3.14159265359 / 180);
		def round(precision):.*pow(10;precision)|round/pow(10;precision);
		def distance(lat1;lon1;lat2;lon2):
			pow(to_radians(lon2)-to_radians(lon1);2)+pow(to_radians(lat2)-to_radians(lat1);2)|sqrt|. * 6371;
	[.message.list[]|
		select(.name | contains("Costco") | not)|
		{
			"price":.prices.U91.amount,
			"updated":((now-(.prices.U91.updated/1000))/3600|floor),
			"name":.name,
			"distance":distance(.location.y;.location.x;%0.14f;%0.14f)|round(1)
		}
		|select(.updated < 24)
	]|
	[
		sort_by(.distance)|.[0:5]|.[]|"\(.price)@\(.name) (\(.updated)h, \(.distance)km)"
	]|join("\n")`, lat, lon)

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

		//lint:ignore SA4004 easiest way to get result as string
		return fmt.Sprint(v)
	}
	return ""
}
