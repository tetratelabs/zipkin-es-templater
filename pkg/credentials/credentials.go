package credentials

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
)

// ReadFile is used to extract user/pass credentials for connecting to ES from a file using the provided user/pass
// extraction paths.
func ReadFile(fileName, tplUser, tplPass string) (user string, pass string, err error) {
	var (
		b   []byte
		obj interface{}
	)

	b, err = ioutil.ReadFile(fileName)
	if err != nil {
		return "", "", err
	}

	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".yml", ".yaml":
		obj, err = deserializeYAML(b)
		if err != nil {
			return
		}
	case ".json":
		obj, err = deserializeJSON(b)
		if err != nil {
			return
		}
	default:
		obj, err = deserializeKV(b)
		if err != nil {
			return
		}
		if tplUser == "" {
			tplUser = "{{ .username }}"
		}
		if tplPass == "" {
			tplPass = "{{ .password }}"
		}
	}

	if tplUser == "" {
		tplUser = "{{ .data.username }}"
	}

	if tplPass == "" {
		tplPass = "{{ .data.password }}"
	}

	user, err = extractor(obj, tplUser)
	if err != nil {
		return "", "", err
	}

	pass, err = extractor(obj, tplPass)
	if err != nil {
		return "", "", err
	}

	return user, pass, nil
}

// extractor extracts a value from object based on the provided template.
func extractor(obj interface{}, tpl string) (string, error) {
	var b bytes.Buffer

	t, err := template.New("tpl").Parse(tpl)
	if err != nil {
		return "", err
	}

	if err = t.Execute(&b, obj); err != nil {
		return "", err
	}

	return b.String(), nil
}

func deserializeJSON(b []byte) (interface{}, error) {
	var data interface{}

	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func deserializeYAML(b []byte) (interface{}, error) {
	var data interface{}

	if err := yaml.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func deserializeKV(b []byte) (interface{}, error) {
	data := make(map[string]string)

	for _, line := range bytes.Split(b, []byte("\n")) {
		kv := bytes.SplitN(line, []byte("="), 2)
		if len(kv) == 2 {
			data[string(bytes.TrimSpace(kv[0]))] = string(bytes.TrimSpace(kv[1]))
		} else if len(bytes.TrimSpace(kv[0])) > 0 {
			log.Printf("unable to parse <key>=<value> pair for: %s", string(kv[0]))
		}
	}

	if len(data) == 0 {
		return nil, errors.New("no key:value pairs found")
	}

	return data, nil
}
