package es

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tetratelabs/zipkin-es-templater/pkg/templater"
)

// ClusterInfo response
type ClusterInfo struct {
	Name        string `json:"name"`
	ClusterName string `json:"cluster_name"`
	ClusterUUID string `json:"cluster_uuid"`
	Version     struct {
		Number                           string    `json:"number"`
		BuildFlavor                      string    `json:"build_flavor"`
		BuildType                        string    `json:"build_type"`
		BuildHash                        string    `json:"build_hash"`
		BuildDate                        time.Time `json:"build_date"`
		BuildSnapshot                    bool      `json:"build_snapshot"`
		LuceneVersion                    string    `json:"lucene_version"`
		MinimumWireCompatibilityVersion  string    `json:"minimum_wire_compatibility_version"`
		MinimumIndexCompatibilityVersion string    `json:"minimum_index_compatibility_version"`
	} `json:"version"`
	Tagline string `json:"tagline"`
}

// Client holds an ES client for Zipkin specific ES management.
type Client struct {
	client  *http.Client
	host    string
	ci      ClusterInfo
	version float64
}

// NewClient returns a new Zipkin specific ES management Client.
func NewClient(client *http.Client, host string) (*Client, error) {
	if client == nil {
		client = http.DefaultClient
	}

	c := Client{
		client: client,
		host:   host,
	}

	ci, err := c.getClusterInfo()
	if err != nil {
		return nil, err
	}
	c.ci = *ci

	if c.version, err = c.parseVersion(); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *Client) getClusterInfo() (*ClusterInfo, error) {
	res, err := c.client.Get(c.host)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var ci ClusterInfo
	if err = json.NewDecoder(res.Body).Decode(&ci); err != nil {
		return nil, err
	}

	return &ci, nil
}

func (c Client) parseVersion() (float64, error) {
	v := strings.Split(c.ci.Version.Number, ".")
	if len(v) != 3 {
		return 0.0, errors.New("invalid version number")
	}
	return strconv.ParseFloat(v[0]+"."+v[1], 64)
}

// Version returns the ES version of the registered ES host.
func (c Client) Version() float64 {
	return c.version
}

// SetIndexTemplate tries to insert provided template
func (c Client) SetIndexTemplate(templateName string, tpl templater.Template) (string, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(tpl); err != nil {
		return "", err
	}
	req, err := http.NewRequest("PUT", c.host+"/_template/"+templateName, buf)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// GetTemplates returns templates given provided template pattern.
func (c Client) GetTemplates(tplPattern string) (map[string]templater.Template, error) {
	res, err := c.client.Get(c.host + "/_template/" + tplPattern + "?local=false")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 && res.StatusCode != 404 {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(b))
	}
	tpls := make(map[string]templater.Template)
	if err = json.NewDecoder(res.Body).Decode(&tpls); err != nil {
		return nil, err
	}
	return tpls, nil
}
