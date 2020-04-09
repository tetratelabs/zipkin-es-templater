# zipkin-es-templater
Tests for and creates when needed Elasticsearch index templates for Zipkin

Command line arguments:
```bash

Usage of templater settings:
      --ca-bundle string              ca-bundle for self signed https
      --disable-search                disable search indexes (if not using Zipkin UI)
      --disable-strict-traceId        disable strict traceID (when migrating between 64-128bit)
      --es-password string            basic auth password
      --es-username string            basic auth username
  -H, --host string                   Elasticsearch host URL (default "http://localhost:9200")
      --log-as-json                   Whether to format output as JSON or in plain console-friendly format
      --log-caller string             Comma-separated list of scopes for which to include called information, scopes can be any of [default]
      --log-output-level string       The minimum logging level of messages to output,  can be one of [debug, info, warn, error, none] (default "default:info")
      --log-rotate string             The path for the optional rotating log file
      --log-rotate-max-age int        The maximum age in days of a log file beyond which the file is rotated (0 indicates no limit) (default 30)
      --log-rotate-max-backups int    The maximum number of log file backups to keep before older files are deleted (0 indicates no limit) (default 1000)
      --log-rotate-max-size int       The maximum size in megabytes of a log file beyond which the file is rotated (default 104857600)
      --log-stacktrace-level string   The minimum logging level at which stack traces are captured, can be one of [debug, info, warn, error, none] (default "default:none")
      --log-target stringArray        The set of paths where to output the log. This can be any path as well as the special values stdout and stderr (default [stdout])
  -p, --prefix string                 index template name prefix (default "zipkin")
      --purge-data                    purge existing Zipkin data (useful if incorrectly indexed)
  -r, --replicas int                  index replica count (default 1)
  -s, --shards int                    index shard count (default 5)

```

Environment Variables:

```bash
INDEX_PREFIX=zipkin \
    INDEX_REPLICAS=1 \
    INDEX_SHARDS=5 \
    ES_HOST="https://localhost:9200" \
    DISABLE_STRICT_TRACEID=0 \
    DISABLE_SEARCH=0 \
    ./zipkin-es-templater
```
