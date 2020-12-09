package fireside

import (
    "encoding/json"
    "time"
)

type FiresideEvent struct {
    Event  FiresideEventEvent  `json:"event"`
    Source FiresideEventSource `json:"source"`
}

type FiresideEventEvent struct {
    Category  string `json:"category"`
    // 'Data' field is always empty in struct; event 'Data' is inserted after JSON conversion
    //Data     map[string]interface{}
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
    event["raw"] = dataMap
    eventJson, err := json.Marshal(event)
    if err != nil { return nil, err }

    return eventJson, nil
}
