package fireside

import (
    "encoding/json"
    "time"

    log "github.com/sirupsen/logrus"
)

type FiresideEvent struct {
    // 'Data' field is always empty in struct; event.data is inserted
    // after JSON conversion and contains the raw JSON for the source event.
    Data   map[string]interface{} `json:"data"`
    // 'Event' field contains a map of other fields that provide context
    // regarding the category, time, type, etc. of the event.
    Event  FiresideEventEvent     `json:"event"`
    // 'Source' filed contains a map of other fields that provide context
    // regarding how the event was sourced.
    Source FiresideEventSource    `json:"source"`
}

type FiresideEventEvent struct {
    Category  string `json:"category"`
    Timestamp string `json:"timestamp"`
    Type      string `json:"type"`
}

// data related to how and where the event was sourced
type FiresideEventSource struct {
    ID   string `json:"id"`
    Path string `json:"path"`
    Type string `json:"type"`
}

func NewFiresideEvent(eventCat string, eventType string, srcID string, srcPath string, srcType string) *FiresideEvent {
    return &FiresideEvent{
        Event: FiresideEventEvent{
            Category: eventCat,
            Timestamp: time.Now().Format(time.RFC3339),
            Type: eventType,
        },
        Source: FiresideEventSource{
            ID: srcID,
            Path: srcPath,
            Type: srcType,
        },
    }
}

func InsertFiresideEventData(eventData []byte, eventWrapper *FiresideEvent) ([]byte, error) {
    // unmarshal the JSON for the source entry so that we can insert into
    // a common event wrapper
    var dataMap map[string]interface{}
    derr := json.Unmarshal(eventData, &dataMap)
    if derr != nil { return nil, derr }

    // convert the event wrapper struct to JSON as prep for event data insertion
    eventWrapperJson, werr := json.Marshal(eventWrapper)
    if werr != nil { return nil, werr }

    // unmarshal the JSON for the event wrapper so that we can manipulate the data
    var event map[string]interface{}
    merr := json.Unmarshal(eventWrapperJson, &event)
    if merr != nil { return nil, merr }

    // insert the source event data into the event wrapper
    event["data"] = dataMap
    eventJson, err := json.Marshal(event)
    if err != nil { return nil, err }

    // debug logging
    //log.Debugf("creating FiresideEvent : event.event.category = %s : event.event.source = %s", event.event.category, event.event.source)
    log.Debug(string(eventJson))

    return eventJson, nil
}
