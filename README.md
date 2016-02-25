## Synopsis

Logsearch - query an elasticsearch server for logs produced by logstash-logback module <https://github.com/logstash/logstash-logback-encoder/tree/logstash-logback-encoder-4.6>

Output from query will be in JSON, and it is recommended that the __jq__ tool be used for formatting and extracting information.

Several commandline flags are supported to query specific fields, but arbitrary search terms can be passed to the elasticsearch interface on the commandline. Note elasticsearch query syntax must be followed (specifically AND and OR must be capitalized)

## Usage Example

Logsearch looks for the startup configuration in ~/.logsearch_profile, formatted in TOML ( <https://npf.io/2014/08/intro-to-toml/> ). Any field that is in the Config struct can be preset in the startup profile.

Here is an example file that shows setting the username and password for authenticating to the elasticsearch endpoint:

    $ cat ~/.logsearch_profile
    Username="sychan"
    Password="steves awesome password"
    

__Get usage info__

    logsearch -h
    Usage of logsearch:
      -bundle string
            bundle.name to match
      -config string
            Location of TOML formatted configuration file https://github.com/toml-lang/toml
            NOTE: setting non-default configfile override comflags
            (default "~/.logsearch_profile")
      -context string
            camel.contextId to match
      -correlation string
            camel.correlationID to match
      -count
            Return only a count of the number of matches
      -debug
            Output debug information
      -errors
            Return only error logs that match
      -from string
            Time specification for starting time of the search
            https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-date-format.html
            (default "now-30m")
      -logger string
            logger_name to match
      -message string
            Match against the main log message
      -numlogs int
            Number of lines of matching logs to return at a time (default 100)
      -offset int
            Offset into total matching logs to start
      -password string
            Password for basic auth (default "gimmedata,now!")
      -stack string
            stack_trace to match
      -uat
            Search among the UAT logs for matches
      -until string
            Time specification for ending time of search (default "now")
      -url string
            Base URL for the elasticsearch server (default "http://127.0.0.1:9200/")
      -username string
            Enable basic auth by setting username (default "analytics")

__Get a count of the number of prod ESB errors in the last 30 minutes (count is in the "count" attribute)__

    logsearch -errors -count | jq .
    {
      "_shards": {
        "failed": 0,
        "successful": 261,
        "total": 261
      },
      "count": 1
    }
 
__Error count for the last 6 hours__

    logsearch -errors -count -from now-6h | jq .
    {
      "_shards": {
        "failed": 0,
        "successful": 261,
        "total": 261
      },
      "count": 87
    }

__Get a count of the number of errors in the last 6 hours in the UAT logs__

    logsearch -errors -count -from now-6h -uat | jq .
    {
      "_shards": {
        "failed": 0,
        "successful": 261,
        "total": 261
      },
      "count": 419
    }

__Get first 3 errors in the last 6 hours in the logs__

    logsearch -errors -from now-6h -numlogs 3 | jq .
    {
      "took": 94,
      "timed_out": false,
      "_shards": {
        "total": 266,
        "successful": 261,
        "failed": 0
      },
      "hits": {
        "total": 30,
        "max_score": null,
        "hits": [
          {
           "_index": "logstash-2016.02.18",
           "_type": "logstash_tcp",
           "_id": "AVL1dQKxm3bUyDWoQRTM",
           "_score": null,
           "_source": {
             "@timestamp": "2016-02-18T16:19:44.734Z",
             "@version": 1,
             "message": "Failed delivery for (MessageId: ID-api-esb-prod-03-ist-berkeley-edu-47910-1455806566968-18-48 on ExchangeId: ID-api-esb-prod-03-ist-berkeley-edu-47910-1455806566968-18-46). Exhausted after delivery attempt: 1 caught: java.lang.Exception: Fetching array element 1: index is not in range 1 to 0. (180,252) UC_PKG.UC_CC_Handler.UC_C
      [ truncated for brevity ]

__Get a count of log entries where "Broker" appears in the message field in the last 24h:__

Note that we can explicitly pass query parameters to the elasticsearch backend by simply making them non-flag arguments to the program.

    logsearch -from now-24h -count message:Broker | jq .
    {
      "count": 44,
      "_shards": {
        "total": 266,
        "successful": 261,
        "failed": 0
      }
    }

__Get a count of log entries where "Broker" appears in the message field in the last 24h from the host api-esb-prod-03. Note the use of backslashes to escape \" when passed as an explicit search term (needed for the shell):__

We can add multiple search terms, but they require an explicit AND/OR between them, and they need to be capitalized. The search interface also tokenizes at non-alphanumeric characters, so the - in api-esb-prod-03 needs to be quoted so that the final search term is HOSTNAME:\"api-esb-prod-03\". This requires double escaping to get past the shell.

    logsearch -from now-24h -count message:Broker AND HOSTNAME:\\\"api-esb-prod-03\\\" | jq .
    {
      "count": 32,
      "_shards": {
        "total": 266,
        "successful": 261,
        "failed": 0
      }
    }

## Motivation

Commandline client to query the ElasticSearch backend, bypassing Kibana

