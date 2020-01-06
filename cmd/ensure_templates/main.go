package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	l "github.com/tetratelabs/log"

	"github.com/tetratelabs/zipkin-es-templater/pkg/es"
	t "github.com/tetratelabs/zipkin-es-templater/pkg/templater"
)

const (
	templatePath = "/_template/"
)

var (
	log     = l.RegisterScope("default", "Ensure Zipkin ES Index Templates", 0)
	logOpts = l.DefaultOptions()
)

func main() {
	// init our defaults
	var settings = struct {
		t.VersionSpecificTemplates
		host                 string
		disableStrictTraceID bool
		disableSearch        bool
		waitForActiveShards  int
	}{
		VersionSpecificTemplates: t.VersionSpecificTemplates{
			IndexPrefix:   "zipkin",
			IndexReplicas: 1,
			IndexShards:   5,
		},
		host: "http://localhost:9200",
	}
	// os env override
	{
		if str := os.Getenv("INDEX_PREFIX"); str != "" {
			settings.IndexPrefix = str
		}
		if str := os.Getenv("INDEX_REPLICAS"); str != "" {
			if i, err := strconv.ParseInt(str, 10, 32); err == nil {
				settings.IndexReplicas = int(i)
			}
		}
		if str := os.Getenv("INDEX_SHARDS"); str != "" {
			if i, err := strconv.ParseInt(str, 10, 32); err == nil {
				settings.IndexShards = int(i)
			}
		}
		if str := os.Getenv("ES_HOST"); str != "" {
			settings.host = str
		}
		if str, found := os.LookupEnv("DISABLE_STRICT_TRACEID"); found {
			str = strings.ToLower(str)
			if str == "1" || str == "yes" || str == "on" {
				settings.disableStrictTraceID = true
			}
		}
		if str, found := os.LookupEnv("DISABLE_SEARCH"); found {
			str = strings.ToLower(str)
			if str == "1" || str == "yes" || str == "on" {
				settings.disableStrictTraceID = true
			}
		}
	}

	// flag handling
	{
		fs := pflag.NewFlagSet("templater settings", pflag.ContinueOnError)
		fs.StringVarP(&settings.IndexPrefix, "prefix", "p",
			settings.IndexPrefix, "index template name prefix")
		fs.IntVarP(&settings.IndexReplicas, "replicas", "r",
			settings.IndexReplicas, "index replica count")
		fs.IntVarP(&settings.IndexShards, "shards", "s", settings.IndexShards,
			"index shard count")
		fs.BoolVar(&settings.disableStrictTraceID, "disable-strict-traceId",
			settings.disableStrictTraceID,
			"disable strict traceID (when migrating between 64-128bit)")
		fs.BoolVar(&settings.disableSearch, "disable-search",
			settings.disableSearch,
			"disable search indexes (if not using Zipkin UI)")
		fs.StringVarP(&settings.host, "host", "H", settings.host,
			"Elasticsearch host URL")

		logOpts.AttachToFlagSet(fs)

		// parse FlagSet and exit on error
		if err := fs.Parse(os.Args[1:]); err != nil {
			if err == pflag.ErrHelp {
				os.Exit(0)
			}
			fmt.Printf("unable to parse settings: %+v\n", err)
			os.Exit(1)
		}

		settings.StrictTraceID = !settings.disableStrictTraceID
		settings.SearchEnabled = !settings.disableSearch
	}

	// initialize the logging subsystem
	if err := l.Configure(logOpts); err != nil {
		fmt.Printf("failed to configure logging: %v\n", err)
		os.Exit(1)
	}

	//	create our http client
	client := &http.Client{}

	log.Debugf("trying to connect to host: %s", settings.host)

	ci, err := es.GetClusterInfo(client, settings.host)
	if err != nil {
		fmt.Printf("unable to retrieve cluster info: %+v\n", err)
		os.Exit(1)
	}

	settings.Version, err = es.ExtractVersion(ci)
	if err != nil {
		log.Errorf("unable to retrieve ES version: %+v", err)
		os.Exit(1)
	}

	log.Infof("connected to Elasticsearch version: %g", settings.Version)

	res, err := http.Get(settings.host + templatePath + settings.IndexPrefix +
		t.IndexTypeDelimiter(settings.Version) + "*?local=false")
	if err != nil {
		log.Errorf("unable to check templates: %+v", err)
		os.Exit(1)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 && res.StatusCode != 404 {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Errorf("unable to check templates: %+v", err)
			os.Exit(1)
		}
		log.Errorf("unable to check templates: %s", string(b))
		os.Exit(1)
	}
	tpls := make(map[string]t.Template)
	if err = json.NewDecoder(res.Body).Decode(&tpls); err != nil {
		log.Errorf("unable to deserialize template check: %+v", err)
		os.Exit(1)
	}

	// check for SpanIndexTemplate
	key := settings.IndexPrefix + t.IndexTypeDelimiter(settings.Version) +
		t.SpanType + t.TemplateSuffix
	if _, found := tpls[key]; !found {
		log.Infof("spanIndex %q template missing", key)
		tpl, err := t.SpanIndexTemplate(settings.VersionSpecificTemplates)
		if err != nil {
			log.Errorf("unable to generate spanIndexTemplate: %+v", err)
			os.Exit(1)
		}

		res, err := es.SetIndexTemplate(
			client, settings.host+"/_template/"+settings.IndexPrefix+"-"+
				t.SpanType+"_template", tpl,
		)
		if err != nil {
			log.Errorf("unable to create spanIndexTemplate: %+v", err)
			os.Exit(1)
		}
		log.Infof("spanIndex: %s", res)
	} else {
		log.Debugf("spanIndex template found")
	}

	// check for AutoCompleteTemplate
	key = settings.IndexPrefix + t.IndexTypeDelimiter(settings.Version) +
		t.AutoCompleteType + t.TemplateSuffix
	if _, found := tpls[key]; !found {
		log.Infof("autoComplete %q template missing", key)
		tpl, err := t.AutoCompleteTemplate(settings.VersionSpecificTemplates)
		if err != nil {
			log.Errorf("unable to generate autoCompleteTemplate: %+v", err)
			os.Exit(1)
		}

		res, err := es.SetIndexTemplate(
			client, settings.host+"/_template/"+settings.IndexPrefix+"-"+
				t.AutoCompleteType+"_template", tpl,
		)
		if err != nil {
			log.Errorf("unable to create autoCompleteTemplate: %+v", err)
			os.Exit(1)
		}
		log.Infof("autoComplete: %s", res)
	} else {
		log.Debugf("autoComplete template found")
	}

	// check for DependencyTemplate
	key = settings.IndexPrefix + t.IndexTypeDelimiter(settings.Version) +
		t.DependencyType + t.TemplateSuffix
	if _, found := tpls[key]; !found {
		log.Infof("dependency %q template missing", key)
		tpl, err := t.DependencyTemplate(settings.VersionSpecificTemplates)
		if err != nil {
			log.Errorf("unable to generate dependencyTemplate: %+v", err)
			os.Exit(1)
		}

		res, err := es.SetIndexTemplate(
			client, settings.host+"/_template/"+settings.IndexPrefix+"-"+
				t.DependencyType+"_template", tpl,
		)
		if err != nil {
			log.Errorf("unable to create dependencyTemplate: %+v", err)
			os.Exit(1)
		}
		log.Infof("dependencyTemplate: %s", res)
	} else {
		log.Debugf("dependency template found")
	}

}
