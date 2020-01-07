// Copyright 2020 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License
// is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied. See the License for the specific language governing permissions and limitations under
// the License.

// Package templater contains logic to generate version specific elasticsearch index templates.
package templater

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// IndexTemplateType for Zipkin indexes.
type IndexTemplateType string

// constants
const (
	// maximum character length constraint of most names, IP literals and IDs
	shortStringLength = 256
	TemplateSuffix    = "_template"

	AutoCompleteType IndexTemplateType = "autocomplete"
	SpanType         IndexTemplateType = "span"
	DependencyType   IndexTemplateType = "dependency"
)

var (
	_false = false
	_true  = true
	// in Zipkin search, we do exact match only (keyword). Norms is about
	// scoring. We don't use that in our API, and disable it to reduce disk
	// storage needed.
	keyWord = Field{Type: "keyword", Norms: &_false}
)

// Config holds the configuration data for a Service.
type Config struct {
	IndexPrefix   string
	IndexReplicas int
	IndexShards   int
	SearchEnabled bool
	StrictTraceID bool
}

// DefaultConfig returns a Config object with default settings initialized.
func DefaultConfig() Config {
	return Config{
		IndexPrefix:   "zipkin",
		IndexReplicas: 1,
		IndexShards:   5,
		SearchEnabled: true,
		StrictTraceID: true,
	}
}

// Service allows to construct ES version specific index templates for Zipkin.
type Service struct {
	cfg                Config
	version            float64
	indexTypeDelimiter string
}

// New returns a templating Service configured to the provided config values and
// ES version.
func New(config Config, version float64) (*Service, error) {
	if version < 5.0 || version >= 8 {
		return nil, fmt.Errorf(
			"Elasticsearch versions 5-7.x are supported, was: %f", version)
	}

	s := Service{
		cfg:                config,
		version:            version,
		indexTypeDelimiter: "-",
	}

	// IndexTypeDelimiter returns a delimiter based on what's supported by the
	// Elasticsearch version.
	// Starting in Elasticsearch 7.x, colons are no longer allowed in index
	// names. This logic will make sure the pattern in our index template
	// doesn't use them either.
	// See: https://github.com/openzipkin/zipkin/issues/2219
	if version < 7.0 {
		s.indexTypeDelimiter = ":"
	}

	return &s, nil
}

// TemplateByType returns a generated template for the provided type.
func (s Service) TemplateByType(t IndexTemplateType) *Template {
	var tpl Template
	switch t {
	case AutoCompleteType:
		tpl = s.AutoCompleteTemplate()
	case DependencyType:
		tpl = s.DependencyTemplate()
	case SpanType:
		tpl = s.SpanIndexTemplate()
	default:
		return nil
	}
	return &tpl
}

// SpanIndexTemplate returns a span index template object that satisfies the
// provided Zipkin and ES version specific settings.
func (s Service) SpanIndexTemplate() Template {
	t := Template{Settings: s.indexProperties()}

	t.setIndexName(s.version, s.indexPattern(SpanType))

	traceIDMapping := keyWord

	if !s.cfg.StrictTraceID {
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

	if s.cfg.SearchEnabled {
		m = Mappings{
			Source: &MetaField{Excludes: []string{"_q"}},
			DynamicTemplates: []DynamicTemplate{
				{"strings": {
					Mapping: Field{
						Type:        "keyword",
						Norms:       &_false,
						IgnoreAbove: shortStringLength,
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

	t.Mappings = m.AttachToTemplate(SpanType, s.version)

	return t
}

// DependencyTemplate returns a dependency template object that satisfies the
// provided Zipkin and ES version specific settings.
func (s Service) DependencyTemplate() Template {
	t := Template{
		Settings: s.indexProperties(),
	}

	t.setIndexName(s.version, s.indexPattern(DependencyType))

	m := Mappings{
		Enabled: &_false,
	}
	t.Mappings = m.AttachToTemplate(DependencyType, s.version)

	return t
}

// AutoCompleteTemplate returns an autocomplete template object that satisfies
// the provided Zipkin and ES version specific settings.
func (s Service) AutoCompleteTemplate() Template {
	t := Template{
		Settings: s.indexProperties(),
	}

	t.setIndexName(s.version, s.indexPattern(AutoCompleteType))

	m := Mappings{
		Enabled: &_true,
		Properties: map[string]Field{
			"tagKey":   keyWord,
			"tagValue": keyWord,
		},
	}
	t.Mappings = m.AttachToTemplate(AutoCompleteType, s.version)

	return t
}

func (s Service) indexProperties() Settings {
	// 6.x _all disabled https://www.elastic.co/guide/en/elasticsearch/reference/6.7/breaking-changes-6.0.html#_the_literal__all_literal_meta_field_is_now_disabled_by_default
	// 7.x _default disallowed https://www.elastic.co/guide/en/elasticsearch/reference/current/breaking-changes-7.0.html#_the_literal__default__literal_mapping_is_no_longer_allowed
	settings := Settings{
		Index: Index{
			NumberOfReplicas:    strconv.Itoa(s.cfg.IndexReplicas),
			NumberOfShards:      strconv.Itoa(s.cfg.IndexShards),
			RequestsCacheEnable: true,
		},
	}
	if s.version < 7.0 {
		// there is no explicit documentation of index.mapper.dynamic being
		// removed in v7, but it was.
		settings.Index.MapperDynamic = &_false
	}
	return settings
}

func (s Service) indexPattern(typ IndexTemplateType) string {
	return s.cfg.IndexPrefix + s.indexTypeDelimiter + string(typ) + "-*"
}

// IndexPrefix returns the index prefix with the ES version specific index type
// delimiter.
func (s Service) IndexPrefix() string {
	return s.cfg.IndexPrefix + s.indexTypeDelimiter
}

// IndexTemplateKey returns the fully named key for indexTypeName
func (s Service) IndexTemplateKey(indexTypeName IndexTemplateType) string {
	return s.cfg.IndexPrefix + s.indexTypeDelimiter + string(indexTypeName) +
		TemplateSuffix
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
func (t *Template) setIndexName(version float64, name string) {
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
	NumberOfShards      string `json:"number_of_shards,omitempty"`
	NumberOfReplicas    string `json:"number_of_replicas,omitempty"`
	RequestsCacheEnable bool   `json:"requests.cache.enable,omitempty"`
	MapperDynamic       *bool  `json:"mapper.dynamic,omitempty"`
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
func (m Mappings) AttachToTemplate(name IndexTemplateType, version float64) interface{} {
	// ES 7.x defaults include_type_name to false https://www.elastic.co/guide/en/elasticsearch/reference/current/breaking-changes-7.0.html#_literal_include_type_name_literal_now_defaults_to_literal_false_literal
	if version < 7.0 {
		nm := make(map[string]Mappings)
		nm[string(name)] = m
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
