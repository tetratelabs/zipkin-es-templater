package templater

import (
	"encoding/json"
	"fmt"
)

// defaults
const (
	ShortStringLength = 256
	AutoCompleteType  = "autocomplete"
	SpanType          = "span"
	DependencyType    = "dependency"
)

// convenience values
var (
	_false  = false
	_true   = true
	keyWord = Field{Type: "keyword", Norms: &_false}
)

// VersionSpecificTemplates allows to construct ES version specific index
// templates for Zipkin.
type VersionSpecificTemplates struct {
	IndexPrefix   string
	IndexReplicas int
	IndexShards   int
	SearchEnabled bool
	StrictTraceID bool
	Version       float32
}

// DefaultVersionSpecificTemplates holds default values.
var DefaultVersionSpecificTemplates = VersionSpecificTemplates{
	IndexPrefix:   "zipkin",
	IndexReplicas: 1,
	IndexShards:   5,
	SearchEnabled: true,
	StrictTraceID: true,
	Version:       7.0,
}

// SpanIndexTemplate returns a span index template object that satisfies the
// provided (version specific) settings.
func SpanIndexTemplate(v VersionSpecificTemplates) (*Template, error) {
	if err := testSupportedVersion(v); err != nil {
		return nil, err
	}
	t := Template{Settings: indexProperties(v)}

	t.SetIndexName(v.Version, indexPattern(v, DependencyType))

	traceIDMapping := keyWord

	if !v.StrictTraceID {
		// Supporting mixed trace ID length is expensive due to needing a
		// special analyzer and "fielddata" which consumes a lot of heap. Sites
		// should only turn off strict trace ID when in a transition, and keep
		// trace ID length transitions as short time as possible.
		traceIDMapping = Field{
			Type:      "text",
			Fielddata: &_true,
			Analyzer:  "traceId_analyzer",
		}
		t.Settings.Analysis = &Analysis{
			Analyzer: map[string]Analyzer{
				"traceId_analyzer": {
					Type:      "custom",
					Tokenizer: "keyword",
					Filter:    []string{"traceId_filter"},
				},
			},
			Filter: map[string]Filter{
				"traceId_filter": {
					Type:             "pattern_capture",
					Patterns:         []string{"([0-9a-f]{1,16})$"},
					PreserveOriginal: &_true,
				},
			},
		}
	}
	var m Mappings

	if v.SearchEnabled {
		m = Mappings{
			Source: &MetaField{Excludes: []string{"_q"}},
			DynamicTemplates: []DynamicTemplate{
				{"strings": {
					Mapping: Field{
						Type:        "keyword",
						Norms:       &_false,
						IgnoreAbove: ShortStringLength,
					},
					MatchMappingType: "string",
					Match:            "*",
				}},
			},
			Properties: map[string]Field{
				"traceId": traceIDMapping,
				"name":    keyWord,
				"localEndpoint": {
					Type:       "object",
					Dynamic:    &_false,
					Properties: map[string]Field{"serviceName": keyWord},
				},
				"remoteEndpoint": {
					Type:       "object",
					Dynamic:    &_false,
					Properties: map[string]Field{"serviceName": keyWord},
				},
				"timestamp_millis": {Type: "date", Format: "epoch_millis"},
				"duration":         {Type: "long"},
				"annotations":      {Enabled: &_false},
				"tags":             {Enabled: &_false},
				"_q":               keyWord,
			},
		}
	} else {
		m = Mappings{
			Properties: map[string]Field{
				"traceId":     traceIDMapping,
				"annotations": {Enabled: &_false},
				"tags":        {Enabled: &_false},
			},
		}
	}

	t.Mappings = m.AttachToTemplate(SpanType, v.Version)

	return &t, nil
}

// DependencyTemplate returns a dependency template object that satisfies the
// provided (version specific) settings.
func DependencyTemplate(v VersionSpecificTemplates) (*Template, error) {
	if err := testSupportedVersion(v); err != nil {
		return nil, err
	}
	t := Template{
		Settings: indexProperties(v),
	}

	t.SetIndexName(v.Version, indexPattern(v, DependencyType))

	m := Mappings{
		Enabled: &_false,
	}
	t.Mappings = m.AttachToTemplate(DependencyType, v.Version)

	return &t, nil
}

// AutoCompleteTemplate returns an autocomplete template object that satisfies
// the provided (version specific) settings.
func AutoCompleteTemplate(v VersionSpecificTemplates) (*Template, error) {
	if err := testSupportedVersion(v); err != nil {
		return nil, err
	}

	t := Template{
		Settings: indexProperties(v),
	}

	t.SetIndexName(v.Version, indexPattern(v, AutoCompleteType))

	m := Mappings{
		Enabled: &_true,
		Properties: map[string]Field{
			"tagKey":   keyWord,
			"tagValue": keyWord,
		},
	}
	t.Mappings = m.AttachToTemplate(AutoCompleteType, v.Version)

	return &t, nil
}

func indexProperties(v VersionSpecificTemplates) Settings {
	s := Settings{
		Index: Index{
			NumberOfReplicas:    v.IndexReplicas,
			NumberOfShards:      v.IndexShards,
			RequestsCacheEnable: true,
		},
	}
	if v.Version < 7.0 {
		s.Index.MapperDynamic = &_false
	}
	return s
}

func indexPattern(v VersionSpecificTemplates, typ string) string {
	return v.IndexPrefix + indexTypeDelimiter(v.Version) + typ + "-*"
}

func indexTypeDelimiter(version float32) string {
	if version < 7.0 {
		return ":"
	}
	return "-"
}

func testSupportedVersion(v VersionSpecificTemplates) error {
	if v.Version < 5.0 || v.Version >= 8 {
		return fmt.Errorf(
			"Elasticsearch versions 5-7.x are supported, was: %f", v.Version)
	}
	return nil
}

// Template type
type Template struct {
	Template      string      `json:"template,omitempty"`       // < v6.0
	IndexPatterns []string    `json:"index_patterns,omitempty"` // >= c6.0
	Settings      Settings    `json:"settings"`
	Mappings      interface{} `json:"mappings"`
}

// SetIndexName sets the name of the index to the correct property given the
// provided ES version.
func (t *Template) SetIndexName(version float32, name string) {
	if version < 6.0 {
		t.Template = name
	} else {
		t.IndexPatterns = []string{name}
	}
}

// Serialize returns a serialized Template object.
func (t Template) Serialize(pretty bool) (string, error) {
	var (
		b   []byte
		err error
	)
	if pretty {
		b, err = json.MarshalIndent(t, "", "\t")
	} else {
		b, err = json.Marshal(t)
	}

	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Settings type
type Settings struct {
	Index    Index     `json:"index,omitempty"`
	Analysis *Analysis `json:"analysis,omitempty"`
}

// Index type
type Index struct {
	NumberOfShards      int   `json:"number_of_shards,omitempty"`
	NumberOfReplicas    int   `json:"number_of_replicas,omitempty"`
	RequestsCacheEnable bool  `json:"requests.cache.enable,omitempty"`
	MapperDynamic       *bool `json:"mapper.dynamic,omitempty"`
}

// Analysis type
type Analysis struct {
	Analyzer map[string]Analyzer `json:"analyzer,omitempty"`
	Filter   map[string]Filter   `json:"filter,omitempty"`
}

// Analyzer type
type Analyzer struct {
	Type      string   `json:"type,omitempty"`
	Tokenizer string   `json:"tokenizer,omitempty"`
	Filter    []string `json:"filter,omitempty"`
}

// Filter type
type Filter struct {
	Type             string   `json:"type,omitempty"`
	Patterns         []string `json:"patterns,omitempty"`
	PreserveOriginal *bool    `json:"preserve_original,omitempty"`
}

// Mappings type
type Mappings struct {
	Enabled          *bool             `json:"enabled,omitempty"`
	Source           *MetaField        `json:"_source,omitempty"`
	DynamicTemplates []DynamicTemplate `json:"dynamic_templates,omitempty"`
	Properties       map[string]Field  `json:"properties,omitempty"`
}

// MetaField type
type MetaField struct {
	Excludes []string `json:"excludes,omitempty"`
}

// AttachToTemplate attaches a Mappings object to an Index Template. Given the
// version of ES it will either be a typed mapping (pre 7.0) or untyped one
// (7.0+)
func (m Mappings) AttachToTemplate(name string, version float32) interface{} {
	if version < 7.0 {
		nm := make(map[string]Mappings)
		nm[name] = m
		return nm
	}
	return m
}

// DynamicTemplate type
type DynamicTemplate map[string]struct {
	MatchMappingType string `json:"match_mapping_type,omitempty"`
	Match            string `json:"match,omitempty"`
	Mapping          Field  `json:"mapping"`
}

// Field type
type Field struct {
	Analyzer    string           `json:"analyzer,omitempty"`
	Dynamic     *bool            `json:"dynamic,omitempty"`
	Enabled     *bool            `json:"enabled,omitempty"`
	Fielddata   *bool            `json:"fielddata,omitempty"`
	Format      string           `json:"format,omitempty"`
	IgnoreAbove int              `json:"ignore_above,omitempty"`
	Norms       *bool            `json:"norms,omitempty"`
	Properties  map[string]Field `json:"properties,omitempty"`
	Type        string           `json:"type,omitempty"`
}
