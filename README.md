## Synopsis

Logsearch - query an elasticsearch server for ESB and Jetty logs produced by logstash-logback module <https://github.com/logstash/logstash-logback-encoder/tree/logstash-logback-encoder-4.6> and Nginx logs parsed from syslog.

Output from query will be in JSON, and it is recommended that the __jq__ tool be used for formatting and extracting information. The jq tool offers powerful JSON parsing/manipulation features and will improve the usability of the output dramatically. <https://stedolan.github.io/jq/manual/>

Several commandline flags are supported to query specific fields, but arbitrary search terms can be passed to the elasticsearch interface on the commandline. Note elasticsearch query syntax must be followed (specifically AND and OR must be capitalized)

## Usage Example

Logsearch looks for the startup configuration in ~/.logsearch_profile, formatted in TOML ( <https://npf.io/2014/08/intro-to-toml/> ). Any field that is in the Config struct can be preset in the startup profile.

Here is an example file that shows setting the username and password for authenticating to the elasticsearch endpoint:

    $ cat ~/.logsearch_profile
    Username="sychan"
    Password="steves awesome password"
    

__Get usage info__

    logsearch -h
    Usage of ./logsearch:
      -app_id string
        	filter by API Central app_id in nginx logs
      -bundle string
        	ESB bundle_name to match
      -client_ip string
        	Client IP address making HTTP - only useful for -nginx queries
      -config string
        	Location of TOML formatted configuration file https://github.com/toml-lang/toml
    		NOTE: setting non-default configfile override comflags
    	 	(default "~/.logsearch_profile")
      -context string
        	ESB camel_contextId to match
      -correlation string
        	ESB camel_correlationID to match
      -count
        	Return only a count of the number of matches
      -debug
        	Output debug information
      -endpoint string
        	jetty/nginx API endpoint to match ( warning: imprecise )
      -errors
        	Return only ESB error logs that match
      -from string
        	Time specification for starting time of the search
		https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-date-format.html
    	 	(default "now-30m")
      -jetty
        	Query for jetty instead of ESB logs
      -logger string
        	ESB logger_name to match
      -logsource string
        	Server hostname to match against
      -message string
        	Match against the main log message/original access log message
      -nginx
        	Query for nginx instead of ESB or Jetty logs
      -numlogs int
        	Number of lines of matching logs to return at a time (default 100)
      -offset int
        	Offset into total matching logs to start
      -password string
        	Password for basic auth (default "[deleted]")
      -stack string
        	ESB stack_trace to match
      -status int
        	jetty/nginx HTTP Return Status code to match
      -uat
        	Search among the esb UAT logs for matches - only available for esb logs
      -until string
        	Time specification for ending time of search (default "now")
      -uri string
        	jetty/nginx uri to match
      -url string
        	Base URL for the elasticsearch server (default "[deleted]")
      -username string
        	Enable basic auth by setting username (default "[deleted]")

__Get a count of the number of jetty requests that resulted in a HTTP 404 Error in the last 30 minutes (count is in the "count" attribute)__

    logsearch -jetty -status 404 -count | jq .
    {
      "count": 9,
      "_shards": {
        "total": 126,
        "successful": 126,
        "failed": 0
      }
    }
    
    # Have jq extract the count field for us
    logsearch -jetty -status 404 -count | jq .count
    9
    
__Error count for the last 6 hours__

    logsearch -jetty -status 500 -count -from now-6h | jq .
    {
      "_shards": {
        "failed": 0,
        "successful": 261,
        "total": 261
      },
      "count": 87
    }

__Get first 3 500 errors in the last 6 hours in the logs__

    logsearch -jetty -status 500 -from now-6h -numlogs 3 | jq .

    {
      "took": 229,
      "timed_out": false,
      "_shards": {
        "total": 121,
        "successful": 121,
        "failed": 0
      },
      "hits": {
        "total": 5,
        "max_score": null,
        "hits": [
          {
            "_index": "logstash-2016.04.29",
            "_type": "logstash_tcp",
            "_id": "AVRfeEjO-fk-eT-z9JZm",
            "_score": null,
            "_source": {
              "@timestamp": "2016-04-29T00:43:24.507Z",
              "@version": 1,
              "@message": "127.0.0.1 - - [2016-04-28T17:43:24.507-07:00] \"GET /cxf/sis/v1/classes/sections?catalog-number=192E&subject-area-code=UGIS&term-id=2168 HTTP/1.0\" 500 80977",
              "HOSTNAME": "api-esb-prod-04.ist.berkeley.edu",
              "host": "128.32.249.78",
              "port": 34629,
              "type": "logstash_tcp",
              "logsource": "api-esb-prod-04.ist.berkeley.edu",
              "@fields_method": "GET",
              "@fields_protocol": "HTTP/1.0",
              "@fields_status_code": 500,
              "@fields_requested_url": "GET /cxf/sis/v1/classes/sections?catalog-number=192E&subject-area-code=UGIS&term-id=2168 HTTP/1.0",
              "@fields_requested_uri": "/cxf/sis/v1/classes/sections",
              "@fields_remote_host": "127.0.0.1",
              "@fields_HOSTNAME": "127.0.0.1",
              "@fields_content_length": 80977,
              "@fields_elapsed_time": 4076,
              "endpoint": "sis/v1/classes/sections"
            },
            "sort": [
              1461890604507,
              1461890604507
            ]
          },
   [ truncated ]

__Get jetty logs, but formatted as the original raw access-log entries__

Note that the _jq_ command allows a lot of filtering and formatting options. If you just want to see the @message field printed out in 'raw' form (instead of JSON encoded string) you can use the json field selectors and the -r raw output flag. Here's a query for just the last to log entries, formatted as just the raw @message field, and then as progressively more verbose JSON, up to the original fully JSON response:

    logsearch -jetty -numlogs 2 | jq -r '.hits.hits[]._source."@message"'
    127.0.0.1 - - [2016-04-28T22:22:51.266-07:00] "GET /cxf/sis/v1/students/26671109/affiliation HTTP/1.0" 200 799
    127.0.0.1 - - [2016-04-28T22:22:51.056-07:00] "GET /cxf/sis/v1/students/26671109/demographic HTTP/1.0" 200 1894

Without the -r flag to jq:

    logsearch -jetty -numlogs 2 | jq '.hits.hits[]._source."@message"' > /tmp/output
    "127.0.0.1 - - [2016-04-28T22:24:15.462-07:00] \"HEAD /cxf/ HTTP/1.0\" 200 6088"
    "127.0.0.1 - - [2016-04-28T22:24:15.455-07:00] \"HEAD /cxf/ HTTP/1.0\" 200 6088"

Without the @message field selector:

    logsearch -jetty -numlogs 2 | jq '.hits.hits[]._source'
    {
      "@timestamp": "2016-04-29T05:26:14.443Z",
      "@version": 1,
      "@message": "127.0.0.1 - - [2016-04-28T22:26:14.443-07:00] \"GET /cxf/sis/v1/students/3031944800/affiliation HTTP/1.0\" 200 772",
      "HOSTNAME": "api-esb-prod-01.ist.berkeley.edu",
      "host": "128.32.249.15",
      "port": 44491,
      "type": "logstash_tcp",
      "logsource": "api-esb-prod-01.ist.berkeley.edu",
      "@fields_method": "GET",
      "@fields_protocol": "HTTP/1.0",
      "@fields_status_code": 200,
      "@fields_requested_url": "GET /cxf/sis/v1/students/3031944800/affiliation HTTP/1.0",
      "@fields_requested_uri": "/cxf/sis/v1/students/3031944800/affiliation",
      "@fields_remote_host": "127.0.0.1",
      "@fields_HOSTNAME": "127.0.0.1",
      "@fields_content_length": 772,
      "@fields_elapsed_time": 122,
      "endpoint": "sis/v1/students"
    }
    {
      "@timestamp": "2016-04-29T05:26:14.305Z",
      "@version": 1,
      "@message": "127.0.0.1 - - [2016-04-28T22:26:14.305-07:00] \"GET /cxf/sis/v1/students/21115475/affiliation HTTP/1.0\" 200 1354",
      "HOSTNAME": "api-esb-prod-01.ist.berkeley.edu",
      "host": "128.32.249.15",
      "port": 44491,
      "type": "logstash_tcp",
      "logsource": "api-esb-prod-01.ist.berkeley.edu",
      "@fields_method": "GET",
      "@fields_protocol": "HTTP/1.0",
      "@fields_status_code": 200,
      "@fields_requested_url": "GET /cxf/sis/v1/students/21115475/affiliation HTTP/1.0",
      "@fields_requested_uri": "/cxf/sis/v1/students/21115475/affiliation",
      "@fields_remote_host": "127.0.0.1",
      "@fields_HOSTNAME": "127.0.0.1",
      "@fields_content_length": 1354,
      "@fields_elapsed_time": 139,
      "endpoint": "sis/v1/students"
    }

Without the _source selector:

    logsearch -jetty -numlogs 2 | jq '.hits.hits[]'
    {
      "_index": "logstash-2016.04.29",
      "_type": "logstash_tcp",
      "_id": "AVRgfHap-fk-eT-zH0WK",
      "_score": null,
      "_source": {
        "@timestamp": "2016-04-29T05:27:35.443Z",
        "@version": 1,
        "@message": "127.0.0.1 - - [2016-04-28T22:27:35.443-07:00] \"HEAD /cxf/ HTTP/1.0\" 200 6088",
        "HOSTNAME": "api-esb-prod-04.ist.berkeley.edu",
        "host": "128.32.249.78",
        "port": 34629,
        "type": "logstash_tcp",
        "logsource": "api-esb-prod-04.ist.berkeley.edu",
        "@fields_method": "HEAD",
        "@fields_protocol": "HTTP/1.0",
        "@fields_status_code": 200,
        "@fields_requested_url": "HEAD /cxf/ HTTP/1.0",
        "@fields_requested_uri": "/cxf/",
        "@fields_remote_host": "127.0.0.1",
        "@fields_HOSTNAME": "127.0.0.1",
        "@fields_content_length": 6088,
        "@fields_elapsed_time": 0
      },
      "sort": [
        1461907655443,
        1461907655443
      ]
    }
    {
      "_index": "logstash-2016.04.29",
      "_type": "logstash_tcp",
      "_id": "AVRgfHap-fk-eT-zH0WJ",
      "_score": null,
      "_source": {
        "@timestamp": "2016-04-29T05:27:35.441Z",
        "@version": 1,
        "@message": "127.0.0.1 - - [2016-04-28T22:27:35.441-07:00] \"HEAD /cxf/ HTTP/1.0\" 200 6088",
        "HOSTNAME": "api-esb-prod-03.ist.berkeley.edu",
        "host": "128.32.249.77",
        "port": 21234,
        "type": "logstash_tcp",
        "logsource": "api-esb-prod-03.ist.berkeley.edu",
        "@fields_method": "HEAD",
        "@fields_protocol": "HTTP/1.0",
        "@fields_status_code": 200,
        "@fields_requested_url": "HEAD /cxf/ HTTP/1.0",
        "@fields_requested_uri": "/cxf/",
        "@fields_remote_host": "127.0.0.1",
        "@fields_HOSTNAME": "127.0.0.1",
        "@fields_content_length": 6088,
        "@fields_elapsed_time": 0
      },
      "sort": [
        1461907655441,
        1461907655441
      ]
    }

Without the .hits.hits selector ( the full original response ):

    logsearch -jetty -numlogs 2 | jq '.'
    {
      "took": 84,
      "timed_out": false,
      "_shards": {
        "total": 121,
        "successful": 121,
        "failed": 0
      },
      "hits": {
        "total": 12209,
        "max_score": null,
        "hits": [
          {
            "_index": "logstash-2016.04.29",
            "_type": "logstash_tcp",
            "_id": "AVRgfaN5-fk-eT-zH3Cr",
            "_score": null,
            "_source": {
              "@timestamp": "2016-04-29T05:28:52.453Z",
              "@version": 1,
              "@message": "127.0.0.1 - - [2016-04-28T22:28:52.453-07:00] \"GET /cxf/sis/v1/students/3032097524/demographic HTTP/1.0\" 200 2095",
              "HOSTNAME": "api-esb-prod-01.ist.berkeley.edu",
              "host": "128.32.249.15",
              "port": 44491,
              "type": "logstash_tcp",
              "logsource": "api-esb-prod-01.ist.berkeley.edu",
              "@fields_method": "GET",
              "@fields_protocol": "HTTP/1.0",
              "@fields_status_code": 200,
              "@fields_requested_url": "GET /cxf/sis/v1/students/3032097524/demographic HTTP/1.0",
              "@fields_requested_uri": "/cxf/sis/v1/students/3032097524/demographic",
              "@fields_remote_host": "127.0.0.1",
              "@fields_HOSTNAME": "127.0.0.1",
              "@fields_content_length": 2095,
              "@fields_elapsed_time": 255,
              "endpoint": "sis/v1/students"
            },
            "sort": [
              1461907732453,
              1461907732453
            ]
          },
          {
            "_index": "logstash-2016.04.29",
            "_type": "logstash_tcp",
            "_id": "AVRgfaDi-fk-eT-zH3Cj",
            "_score": null,
            "_source": {
              "@timestamp": "2016-04-29T05:28:52.035Z",
              "@version": 1,
              "@message": "127.0.0.1 - - [2016-04-28T22:28:52.035-07:00] \"GET /cxf/sis/v1/students/3032097524/contacts HTTP/1.0\" 200 2226",
              "HOSTNAME": "api-esb-prod-01.ist.berkeley.edu",
              "host": "128.32.249.15",
              "port": 44491,
              "type": "logstash_tcp",
              "logsource": "api-esb-prod-01.ist.berkeley.edu",
              "@fields_method": "GET",
              "@fields_protocol": "HTTP/1.0",
              "@fields_status_code": 200,
              "@fields_requested_url": "GET /cxf/sis/v1/students/3032097524/contacts HTTP/1.0",
              "@fields_requested_uri": "/cxf/sis/v1/students/3032097524/contacts",
              "@fields_remote_host": "127.0.0.1",
              "@fields_HOSTNAME": "127.0.0.1",
              "@fields_content_length": 2226,
              "@fields_elapsed_time": 266,
              "endpoint": "sis/v1/students"
            },
            "sort": [
              1461907732035,
              1461907732035
            ]
          }
        ]
      }
    }
    

__Get a count of log entries where "admissions" appears in the message field in the last 24h:__

    logsearch -jetty -from now-24h  -message admissions -count | jq .count
    44

__Get a count of log entries where "admissions" appears in the message field in the last 24h from the host api-esb-prod-03 (using partial match).__

    logsearch -jetty -from now-24h  -message admissions -count -logsource esb-prod-03 | jq .count
    2354

__Get the log entries for the last hour where "admissions" appears in the endpoint and the status code is neither 200 or 201__

The syntax available with the command line arguments is based on simple AND terms. Sometimes you want to use more complex queries. For those you need to study the search syntax ( use the -debug flag to see the query sent to elasticsearch ) and construct the query by adding arbitrary search terms to the end of the query.

    logsearch -jetty -from now-1h -endpoint admissions NOT \@fields_status_code:200 AND NOT \@fields_status_code:201 | jq -r '.hits.hits[]._source."@message"'
    127.0.0.1 - - [2016-04-29T11:13:17.292-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 415 0
    127.0.0.1 - - [2016-04-29T11:10:06.229-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 415 0
    127.0.0.1 - - [2016-04-29T11:07:26.979-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 415 0
    127.0.0.1 - - [2016-04-29T11:05:52.011-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 415 0
    127.0.0.1 - - [2016-04-29T11:05:13.725-07:00] "GET /cxf/sis/v1/admissions/status/17092025?id-type=student-id HTTP/1.0" 404 250
    127.0.0.1 - - [2016-04-29T11:02:16.339-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 415 0
    127.0.0.1 - - [2016-04-29T11:00:38.314-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 202 402
    127.0.0.1 - - [2016-04-29T11:00:36.820-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 202 402
    127.0.0.1 - - [2016-04-29T11:00:35.232-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 202 402
    127.0.0.1 - - [2016-04-29T11:00:34.332-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 202 402
    127.0.0.1 - - [2016-04-29T11:00:33.063-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 202 401
    127.0.0.1 - - [2016-04-29T11:00:31.339-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 202 402
    127.0.0.1 - - [2016-04-29T11:00:29.989-07:00] "POST /cxf/sis/v1/admissions HTTP/1.0" 202 401

__Query for the first app_id that matches c38eca10 in the last 30 minutes__

    logsearch -numlogs 1 -nginx -app_id c38eca10 | jq .hits.hits[]."_source".message
    169.229.216.107 [03/May/2016:12:15:07 -0700] "GET /myfinaid/87987987/finaid?aidYear=2017 HTTP/1.1" 3scale Service: 1006371747932 -- app_id: c38eca10 -- Usage: "usage[SAIT_MyFinAid]=1" -- Status Code: 200 -- 3scale Status Code: 200 -- 3scale Response Time: 0.361

__Query for the count of app_id that matches c38eca10 in the last 30 minutes__

    logsearch -count -nginx -app_id c38eca10 | jq -r .count
    9358

__Query for the count of requests to a URL containing 'student' in the path with a non 200 status code in the last 24hours__

    logsearch -from now-24h -count -nginx -uri student NOT status:200  | jq .count
    8690

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

    # Have jq extract only the count field for display
    logsearch -errors -count | jq .count
    1

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

    logsearch -errors -count -from now-6h -uat | jq .count
    419

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

__Get the first 3 errors in the last 6 hours, but use the jq formatting options to make it look like an ESB logfile__

The JSON output from the database can be a little hard to parse at times, which is why jq is useful because it provides a powerful language for extracting and formatting output. Here we use an output format that is structured like the ESB logfiles:

    logsearch -from now-6h -numlogs 3 -errors | jq -r '.hits.hits[]._source | "\(."@timestamp") \(.HOSTNAMENONCANON) \(.level) [\(.thread_name)]  \(.logger_name) - \(.message)"'
    2016-04-29T06:21:08.855Z api-esb-prod-04 ERROR [Camel (sis-checklist-client-service-route) thread #246 - seda://processMessage]  org.apache.camel.processor.DefaultErrorHandler - Failed delivery for (MessageId: topic_VirtualTopic.prod.event.provisioning.uid.inbox_ID_registry-p1.calnet.1918.berkeley.edu-49610-1461346171160-1_1_4_1_4766 on ExchangeId: ID-api-esb-prod-04-ist-berkeley-edu-21072-1461859588207-14-196). Exhausted after delivery attempt: 1 caught: java.lang.Exception: studentId is a required parameter
    
    Message History
    ---------------------------------------------------------------------------------------------------------------------------------------
    RouteId              ProcessorId          Processor                                                                        Elapsed (ms)
    [route95           ] [route95           ] [seda://processMessage                                                         ] [        11]
    [route94           ] [to300             ] [seda:processMessage                                                           ] [        11]
    [route95           ] [process86         ] [edu.berkeley.sis.checklist.client.processor.ChecklistGetPreProcessor@4944e9b2 ] [        10]
    
    Exchange
    ---------------------------------------------------------------------------------------------------------------------------------------
    Exchange[
    	Id                  ID-api-esb-prod-04-ist-berkeley-edu-21072-1461859588207-14-196
    	ExchangePattern     InOut
    	Headers             {breadcrumbId=ID-registry-p1-calnet-1918-berkeley-edu-44048-1461346169220-0-24096, CALNET_ID=drkamkar, CamelHttpResponseCode=404, CamelRedelivered=false, CamelRedeliveryCounter=0, JMSCorrelationID=Camel-ID-registry-p1-calnet-1918-berkeley-edu-44048-1461346169220-0-24100, JMSDeliveryMode=2, JMSDestination=topic://VirtualTopic.prod.event.provisioning.uid.inbox, JMSExpiration=0, JMSMessageID=ID:registry-p1.calnet.1918.berkeley.edu-49610-1461346171160-1:1:4:1:4766, JMSPriority=0, JMSRedelivered=false, JMSReplyTo=queue://prod.event.provisioning.uid.inbox.reply, JMSTimestamp=1461910868841, JMSType=null, JMSXGroupID=null, JMSXUserID=null}
    	BodyType            String
    	Body                studentId is a required parameter
    ]
    
    Stacktrace
    ---------------------------------------------------------------------------------------------------------------------------------------
    2016-04-29T06:13:17.233Z api-esb-prod-04 ERROR [Camel (sis-checklist-client-service-route) thread #246 - seda://processMessage]  org.apache.camel.processor.DefaultErrorHandler - Failed delivery for (MessageId: topic_VirtualTopic.prod.event.provisioning.uid.inbox_ID_registry-p1.calnet.1918.berkeley.edu-49610-1461346171160-1_1_4_1_4765 on ExchangeId: ID-api-esb-prod-04-ist-berkeley-edu-21072-1461859588207-14-194). Exhausted after delivery attempt: 1 caught: java.lang.Exception: studentId is a required parameter
    
    Message History
    ---------------------------------------------------------------------------------------------------------------------------------------
    RouteId              ProcessorId          Processor                                                                        Elapsed (ms)
    [route95           ] [route95           ] [seda://processMessage                                                         ] [        17]
    [route94           ] [to300             ] [seda:processMessage                                                           ] [        17]
    [route95           ] [process86         ] [edu.berkeley.sis.checklist.client.processor.ChecklistGetPreProcessor@4944e9b2 ] [        16]
    
    Exchange
    ---------------------------------------------------------------------------------------------------------------------------------------
    Exchange[
    	Id                  ID-api-esb-prod-04-ist-berkeley-edu-21072-1461859588207-14-194
    	ExchangePattern     InOut
    	Headers             {breadcrumbId=ID-registry-p1-calnet-1918-berkeley-edu-44048-1461346169220-0-24091, CALNET_ID=easprec, CamelHttpResponseCode=404, CamelRedelivered=false, CamelRedeliveryCounter=0, JMSCorrelationID=Camel-ID-registry-p1-calnet-1918-berkeley-edu-44048-1461346169220-0-24095, JMSDeliveryMode=2, JMSDestination=topic://VirtualTopic.prod.event.provisioning.uid.inbox, JMSExpiration=0, JMSMessageID=ID:registry-p1.calnet.1918.berkeley.edu-49610-1461346171160-1:1:4:1:4765, JMSPriority=0, JMSRedelivered=false, JMSReplyTo=queue://prod.event.provisioning.uid.inbox.reply, JMSTimestamp=1461910397213, JMSType=null, JMSXGroupID=null, JMSXUserID=null}
    	BodyType            String
    	Body                studentId is a required parameter
    ]
    
    Stacktrace
    ---------------------------------------------------------------------------------------------------------------------------------------
    2016-04-29T05:45:00.778Z api-esb-prod-03 ERROR [Camel (sis-checklist-client-service-route) thread #426 - seda://processMessage]  org.apache.camel.processor.DefaultErrorHandler - Failed delivery for (MessageId: topic_VirtualTopic.prod.event.provisioning.uid.inbox_ID_registry-p1.calnet.1918.berkeley.edu-49610-1461346171160-1_1_4_1_4760 on ExchangeId: ID-api-esb-prod-03-ist-berkeley-edu-34743-1460901611541-23-12433). Exhausted after delivery attempt: 1 caught: java.lang.Exception: studentId is a required parameter
    
    Message History
    ---------------------------------------------------------------------------------------------------------------------------------------
    RouteId              ProcessorId          Processor                                                                        Elapsed (ms)
    [route121          ] [route121          ] [seda://processMessage                                                         ] [         9]
    [route120          ] [to431             ] [seda:processMessage                                                           ] [         9]
    [route121          ] [process116        ] [edu.berkeley.sis.checklist.client.processor.ChecklistGetPreProcessor@3681f23c ] [         9]
    
    Exchange
    ---------------------------------------------------------------------------------------------------------------------------------------
    Exchange[
    	Id                  ID-api-esb-prod-03-ist-berkeley-edu-34743-1460901611541-23-12433
    	ExchangePattern     InOut
    	Headers             {breadcrumbId=ID-registry-p1-calnet-1918-berkeley-edu-44048-1461346169220-0-24066, CamelHttpResponseCode=404, CamelRedelivered=false, CamelRedeliveryCounter=0, JMSCorrelationID=Camel-ID-registry-p1-calnet-1918-berkeley-edu-44048-1461346169220-0-24070, JMSDeliveryMode=2, JMSDestination=topic://VirtualTopic.prod.event.provisioning.uid.inbox, JMSExpiration=0, JMSMessageID=ID:registry-p1.calnet.1918.berkeley.edu-49610-1461346171160-1:1:4:1:4760, JMSPriority=0, JMSRedelivered=false, JMSReplyTo=queue://prod.event.provisioning.uid.inbox.reply, JMSTimestamp=1461908700749, JMSType=null, JMSXGroupID=null, JMSXUserID=null}
    	BodyType            String
    	Body                studentId is a required parameter
    ]
    
    Stacktrace
    ---------------------------------------------------------------------------------------------------------------------------------------
    

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

