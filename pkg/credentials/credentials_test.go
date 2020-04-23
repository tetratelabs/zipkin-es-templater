package credentials_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/tetratelabs/zipkin-es-templater/pkg/credentials"
)

const (
	dataJSON = `{
  "request_id": "82ab8eea-9a07-4682-8cf6-15a21c21843b",
  "lease_id": "database/creds/my-role/6YWVPFq0BIwkIfsepBSY8dn3",
  "lease_duration": 3600,
  "renewable": true,
  "data": {
    "password": "A1a-0Ni9XOQddSDVbmiB",
    "username": "v-root-my-role-cUSnMKfKIbqWbkEhHf3o-1585155604"
  },
  "warnings": null
}`

	dataYAML = `data:
  password: A1a-juUmf0C0hhXcDtU2
  username: v-root-my-role-h8680kE1mThPuPjfDUOj-1585155708
lease_duration: 3600
lease_id: database/creds/my-role/hZZVv5JqbEKSiJF3nswNmHjz
renewable: true
request_id: db1fe564-da3e-ad70-e680-8bf5f0f61e7b
warnings: null`

	dataKVal = `
password = A1a-deXmf0C0hhXcDtW8
username = v-root-my-role-e3880kE1mThPuPjfDUOj-1585155726`
)

func TestReadFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "es-templater")
	if err != nil {
		t.Fatalf("unable to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	for _, item := range []struct {
		credFileName string
		credFileData string
		wantUser     string
		wantPass     string
	}{
		{"creds.json", dataJSON, "v-root-my-role-cUSnMKfKIbqWbkEhHf3o-1585155604", "A1a-0Ni9XOQddSDVbmiB"},
		{"creds.yaml", dataYAML, "v-root-my-role-h8680kE1mThPuPjfDUOj-1585155708", "A1a-juUmf0C0hhXcDtU2"},
		{"creds.kval", dataKVal, "v-root-my-role-e3880kE1mThPuPjfDUOj-1585155726", "A1a-deXmf0C0hhXcDtW8"},
	} {
		if err = ioutil.WriteFile(dir+"/"+item.credFileName, []byte(item.credFileData), 0644); err != nil {
			t.Fatalf("unable to create creds file: %v", err)
		}
		var gotUser, gotPass string
		gotUser, gotPass, err = credentials.ReadFile(dir+"/"+item.credFileName, "", "")
		if err != nil {
			t.Errorf("unable to parse creds file: %v", err)
		}
		if gotUser != item.wantUser {
			t.Errorf("want user: %v, got user: %v", item.wantUser, gotUser)
		}
		if gotPass != item.wantPass {
			t.Errorf("want pass: %v, got pass: %v", item.wantPass, gotPass)
		}
	}
}
