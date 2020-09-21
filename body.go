package main

import (
	"encoding/json"
	"time"
)

const (

	// endDateKey is the JSON key for the body of a request that indicates the end time for the request.
	endDateKey = "dateRangeEnd"

	// exportKey is the JSON key for the body of a request that indicates the request is for exporting to a file type.
	exportKey = "exportType"

	// generalBody is the JSON body for a general request. It is incomplete and must be added to.
	generalBody = `{
  "sortOrder": "ASCENDING",
  "description": "",
  "rounding": false,
  "withoutDescription": false,
  "amountShown": "EARNED",
  "zoomLevel": "WEEK",
  "userLocale": "en_US",
  "customFields": null,
  "summaryFilter": {
    "sortColumn": "GROUP",
    "groups": [
      "PROJECT",
      "TIMEENTRY"
    ]
  }
}`

	// startDateKey is the JSON key for the body of a request that indicates the start time for the request.
	startDateKey = "dateRangeStart"
)

// makeBody makes a JSON request body. The export argument can be left blank if the request is not for exporting data
// to a file.
func makeBody(end time.Time, export string, start time.Time) (body []byte, err error) {

	// Unmarshal the typical body JSON into a Go type.
	general := make(map[string]interface{})
	if err = json.Unmarshal([]byte(generalBody), &general); err != nil {
		return nil, err
	}

	// Assign the dates.
	general[startDateKey] = start.Format("2006-01-02T15:04:05Z")
	general[endDateKey] = end.Format("2006-01-02T15:04:05Z")

	// Check if exporting.
	if export != "" {
		general[exportKey] = export
	}

	// Turn the Go type back into JSON.
	if body, err = json.Marshal(general); err != nil {
		return nil, err
	}

	return body, nil
}
