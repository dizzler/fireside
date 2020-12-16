package configure

// InputConfigFileReader is a struct for storing config for a given
// event input processor for a given src/event 'Type'. Allows for re-using
// a common config struct for various types of event providers that export
// data (for our consumption) to a file. The Src.Path should be readable
// by the FireSide app at runtime.
type InputConfigFileReader struct {
    Name       string `yaml:"name"` // name of the instantiated file reader object
    BufferSize int    `yaml:"buffer_size"` // size of IO buffer to use when LineByLine is false
    Event      struct {
        Category string `yaml:"category"` // event.catecory value to set for events read from Src.Path
        Type     string `yaml:"type"` // event.type value to set for events read from Src.Path
    } `yaml:"event"`
    Gzipped    bool   `yaml:"gzipped"` // is source gzip compressed
    Line       struct {
        ByLine   bool   `yaml:"by_line"` // scan one line at a time, vice reading entire file to buffer
        DataType string `yaml:"data_type"` // e.g. 'json'|'string'
    } `yaml:"line"`
    Src        struct {
        ID   string `yaml:"id"` // src.id value to set for events read from Src.Path
        Path string `yaml:"path"` // src.path from which to read in events
        Type string `yaml:"type"` // e.g. 'file' (reserved for future use / extension)
    } `yaml:"src"`
}
