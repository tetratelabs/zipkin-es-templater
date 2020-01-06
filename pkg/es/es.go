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

// GetClusterInfo returns an ES host's cluster info.
func GetClusterInfo(client *http.Client, host string) (*ClusterInfo, error) {
	res, err := client.Get(host)
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

// ExtractVersion extracts major.minor as float from a ClusterInfo object.
func ExtractVersion(ci *ClusterInfo) (float64, error) {
	v := strings.Split(ci.Version.Number, ".")
	if len(v) != 3 {
		return 0.0, errors.New("invalid version number")
	}
	return strconv.ParseFloat(v[0]+"."+v[1], 64)
}

// SetIndexTemplate tries to insert provided template
func SetIndexTemplate(client *http.Client, url string, tpl *templater.Template) (string, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(tpl); err != nil {
		return "", err
	}
	req, err := http.NewRequest("PUT", url, buf)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
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
