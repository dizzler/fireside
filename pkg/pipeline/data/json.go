package data

import (
    "bytes"
    "encoding/json"
    "fmt"

    log "github.com/sirupsen/logrus"
)

// JSON is the data type that is passed along all data channels.
// Under the covers, JSON is simply a []byte containing JSON data.
type JSON []byte

// NewJSON is a simple wrapper for json.Marshal.
func NewJSON(v interface{}) (JSON, error) {
    d, err := json.Marshal(v)
    if err != nil {
        log.Debug(fmt.Sprintf("data: failure to marshal JSON %+v - error is \"%v\"", v, err.Error()))
        log.Debug(fmt.Sprintf("	Failed val: %+v", v))
    }
    return d, err
}

// decodes JSON []bytes into a go interface{}; includes the option of decoding
// numbers as json.Number (required for OPA processing of JSON)
func DecodeJsonAsInterface(dataBytes JSON, useNumber bool) (i interface{}, e error) {
    dec := json.NewDecoder(bytes.NewReader(dataBytes))

    // OPA evaluation of input data requires that
    // numeric values must be represented as json.Number
    if useNumber {
        dec.UseNumber()
    }

    // decode source JSON into map[string]interface{}
    e = dec.Decode(&i)

    // return the current values for i and e
    return
}

// decodes JSON []bytes into a map[string]interface{}; includes the option of decoding
// numbers as json.Number (required for OPA processing of JSON); useful for returning
// the event wrapper as a map of string keys, where a given (e.g. "data") field may
// be referenced by name and extracted as a go interface{} (for OPA query eval).
func DecodeJsonAsMap(dataBytes JSON, useNumber bool) (i map[string]interface{}, e error) {
    dec := json.NewDecoder(bytes.NewReader(dataBytes))

    // OPA evaluation of input data requires that
    // numeric values must be represented as json.Number
    if useNumber {
        dec.UseNumber()
    }

    // decode source JSON into map[string]interface{}
    e = dec.Decode(&i)

    // return the current values for i and e
    return
}

// ParseJSON is a simple wrapper for json.Unmarshal
func ParseJSON(d JSON, v interface{}) error {
    err := json.Unmarshal(d, v)
    if err != nil {
        log.Debug(fmt.Sprintf("data: failure to unmarshal JSON into %+v - error is \"%v\"", v, err.Error()))
        log.Debug(fmt.Sprintf("	Failed Data: %+v", string(d)))
    }
    return err
}

// ParseJSONSilent won't log output when unmarshaling fails.
// It can be used in cases where failure is expected.
func ParseJSONSilent(d JSON, v interface{}) error {
    return json.Unmarshal(d, v)
}

// ObjectsFromJSON is a helper for parsing JSON into a slice of
// generic maps/objects. The use-case is when a stage is expecting
// to receive either a JSON object or an array of JSON objects, and
// want to deal with it in a generic fashion.
func ObjectsFromJSON(d JSON) ([]map[string]interface{}, error) {
    var objects []map[string]interface{}

    // return if we have null instead of object(s).
    if bytes.Equal(d, []byte("null")) {
        log.Debug("ObjectsFromJSON: received null. Expected object or objects. Skipping.")
        return objects, nil
    }

    var v interface{}
    err := ParseJSON(d, &v)
    if err != nil {
        return nil, err
    }

    // check if we have a single object or a slice of objects
    switch vv := v.(type) {
    case []interface{}:
        for _, o := range vv {
            objects = append(objects, o.(map[string]interface{}))
        }
    case map[string]interface{}:
        objects = []map[string]interface{}{vv}
    case []map[string]interface{}:
        objects = vv
    default:
        err = fmt.Errorf("ObjectsFromJSON: unsupported data type: %T", vv)
        return nil, err
    }

    return objects, nil
}

// JSONFromHeaderAndRows takes the given header and rows of values, and
// turns it into a JSON array of objects.
func JSONFromHeaderAndRows(header []string, rows [][]interface{}) (JSON, error) {
    var b bytes.Buffer
    b.Write([]byte("["))
    for i, row := range rows {
        if i > 0 {
            b.Write([]byte(","))
        }
        b.Write([]byte("{"))
        for j, v := range row {
            if j > 0 {
                b.Write([]byte(","))
            }
            d, err := NewJSON(v)
            if err != nil {
                return nil, err
            }
            headerStr := "null"
            if len(header) > 0 && len(header) > j {
                headerStr = header[j]
            }
            b.Write([]byte(`"` + headerStr + `":` + string(d)))
        }
        b.Write([]byte("}"))
    }
    b.Write([]byte("]"))

    return JSON(b.Bytes()), nil
}
