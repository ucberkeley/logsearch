package main
//
// Client that performs queries against ElasticSearch backend, searching among
// records inserted using the Logstash-Logback encoder
// sychan 2/16/2016
//

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"bytes"
	"crypto/tls"
	"flag"
	"text/template"
	"os"
	"os/user"
	"syscall"
	"regexp"
	"github.com/BurntSushi/toml"
	"golang.org/x/crypto/ssh/terminal"
)

// Struct used for storing query configuration, as well as populating
// the template that follows
type Config struct {
	UrlBase       string   // endpoint for the ElasticSearch REST service
	Username      string   // if set, will enable basic auth authentication to REST
	Password      string   // password to be used with basic auth
	NumLogs       int      // number of log entries to return in query
	FromTime      string   // start time in @timestamp for search window
	UntilTime     string   // ending time in @timestamp for search
	Offset        int      // used with NumLogs for offset when paginating through results
	Count         bool     // perform only a count, do not return all matches
	Uat           bool     // search only the UAT logs
	Errors        bool     // search only among logs with level set to ERROR
	CorrelationID string   // match for the camel.correlationId
	ContextID     string   // match for the camel.contextId
	StackTrace    string   // match for stack_track field
	BundleName    string   // match for bundle.name field
	Logger        string   // match for logger_name field
	Message       string   // match for the message/@message field
	Uri           string   // match for the fields_requested_uri field in jetty logs
	Status        int      // match for the fields_status_code field - http return code  in jetty logs
	Endpoint      string   // match for the endpoint field in jetty logs- API endpoint
	Logsource     string   // match for the logsource field - fqdn of the host generating log
	ExtTerms      []string // array with extra search terms appended to end of query (include AND and OR explicitly)
	Jetty         bool     // query for jetty logs instead of esb
	Nginx         bool     // query for nginx logs instead of esb or jetty
	AppId         string   // match app_id of api central
	ClientIP      string   // match for client IP address
}

// Go template for a query against ElasticSearch, originally based on sample query from
// Kibana dashboard

const reqTemplate = `
{{- define "nginx_query"}}program:nginx{{if .AppId}} AND app_id:{{.AppId}}{{end}}{{if .Uri}} AND uri_path:\"{{.Uri}}\"{{end}}{{if .Status}} AND status:{{.Status}}{{end}}{{if .Endpoint}} AND endpoint:\"{{.Endpoint}}\"{{end}}{{if .ClientIP}} AND remote_addr:{{.ClientIP}}{{end}}{{if .Message}} AND message:\"{{.Message}}\"{{end}}{{end}}

{{- define "jetty_query"}}type:logstash_tcp AND _exists_:\"@fields_requested_url\"{{if .Uri}} AND @fields_requested_uri:\"{{.Uri}}\"{{end}}{{if .Status}} AND @fields_status_code:{{.Status}}{{end}}{{if .Endpoint}} AND endpoint:\"{{.Endpoint}}\"{{end}}{{if .ClientIP}} AND @fields_remote_host:{{.ClientIP}}{{end}}{{if .Message}} AND @message:\"{{.Message}}\"{{end}}{{end}}

{{- define "esb_query"}}program:fuseesb AND app_homedir:\"/home/app_smx{{ if .Uat}}_sg0{{end}}/jboss-fuse-6.1.0.redhat-379\"{{if .Errors}} AND level:\"ERROR\"{{end}}{{if .CorrelationID}} AND camel_correlationId:\"{{.CorrelationID}}\"{{end}}{{if .ContextID}} AND camel_contextId:\"{{.ContextID}}\"{{end}}{{if .StackTrace}} AND stack_trace:\"{{.StackTrace}}\"{{end}}{{if .Logger}} AND logger_name:\"{{.Logger}}\"{{end}}{{if .BundleName}} AND bundle_name:\"{{.BundleName}}\"{{end}}{{if .Message}} AND message:\"{{.Message}}\"{{end}}{{end -}}

{
{{ if not .Count }}  "fields" : [ "_source" ],{{end}}
  "query": {
    "filtered": {
      "query": {
        "bool": {
          "should": [
            {
              "query_string": {
                "query": "{{ if .Nginx -}}
                              {{ template "nginx_query" . -}}
                          {{ else if .Jetty -}}
                              {{ template "jetty_query" . -}}
                          {{ else -}}
                              {{ template "esb_query" . -}}
                          {{ end -}}
                          {{if .Logsource}} AND logsource:\"{{.Logsource}}\"{{end}}{{if .ExtTerms}} AND{{end}}{{range .ExtTerms}} {{.}}{{end}}"
              }
            }
          ]
        }
      },
      "filter": {
        "bool": {
          "must": [
            {
              "range": {
                "@timestamp": {
                  "gte": "{{ .FromTime }}",
                  "lte": "{{ .UntilTime}}"
                }
              }
            }
          ]
        }
      }
    }
  }{{ if not .Count }},
  "size": {{ .NumLogs }},
  "from": {{ .Offset }},
  "sort": [
    {
      "@timestamp": {
        "order": "desc",
        "ignore_unmapped": true
      }
    },
    {
      "@timestamp": {
        "order": "desc",
        "ignore_unmapped": true
      }
    }
  ]
{{end}}
}`

// Global debug flag
var debug = false

// filename for configuration file, as well as location
// of the default configuration file
var configFile string
var configFileDef string

var conf Config

// Initialize the Config struct and commandline parsing flags as well as load in basic
// ~/.logsearch_profile configuration
func init() {
	// Setup some basic configs
	conf.UrlBase = "https://api-devops-prod-02.ist.berkeley.edu/elasticsearch/"
	conf.NumLogs = 100
	conf.FromTime = "now-30m"
	conf.UntilTime = "now"
	
	usr, err := user.Current()
	if err != nil {
		panic( err)
	}
	configFileDef = usr.HomeDir + "/.logsearch_profile"

	// Try to read the default config file, ignore a file not found
	// error
	conf2, err := ReadConfig( conf, configFileDef)
	if err == nil {
		conf = conf2
	} else if os.IsExist(err) {
		fmt.Fprintf(os.Stderr, "Failed to read file: %s\n",configFileDef)
		panic(err)
	}

	// Setup the commandline args parser
	flag.IntVar(&conf.NumLogs, "numlogs", conf.NumLogs, "Number of lines of matching logs to return at a time")
	flag.IntVar(&conf.Offset, "offset", 0, "Offset into total matching logs to start")
	flag.BoolVar(&conf.Count, "count", false, "Return only a count of the number of matches")
	flag.BoolVar(&conf.Errors, "errors", false, "Return only ESB error logs that match")
	flag.BoolVar(&debug, "debug", false, "Output debug information")
	flag.BoolVar(&conf.Jetty, "jetty", false, "Query for jetty instead of ESB logs")
	flag.BoolVar(&conf.Nginx, "nginx", false, "Query for nginx instead of ESB or Jetty logs")
	flag.StringVar(&conf.AppId, "app_id", "", "filter by API Central app_id in nginx logs")
	flag.StringVar(&conf.ClientIP, "client_ip", "", "Client IP address making HTTP - only useful for -nginx queries")
	flag.BoolVar(&conf.Uat, "uat", false, "Search among the esb UAT logs for matches - only available for esb logs")
	flag.StringVar(&conf.FromTime, "from", conf.FromTime, "Time specification for starting time of the search\n\thttps://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-date-format.html\n\t")
	flag.StringVar(&conf.UntilTime, "until", conf.UntilTime, "Time specification for ending time of search")
	flag.StringVar(&conf.CorrelationID, "correlation", "", "ESB camel_correlationID to match")
	flag.StringVar(&conf.ContextID, "context", "", "ESB camel_contextId to match")
	flag.StringVar(&conf.StackTrace, "stack", "", "ESB stack_trace to match")
	flag.StringVar(&conf.BundleName, "bundle", "", "ESB bundle_name to match")
	flag.StringVar(&conf.Logger, "logger", "", "ESB logger_name to match")
	flag.StringVar(&conf.Uri, "uri", "", "jetty/nginx uri to match")
	flag.IntVar(&conf.Status, "status", 0, "jetty/nginx HTTP Return Status code to match")
	flag.StringVar(&conf.Endpoint, "endpoint", "", "jetty/nginx API endpoint to match ( warning: imprecise )")
	flag.StringVar(&conf.Logsource, "logsource", "", "Server hostname to match against")
	flag.StringVar(&conf.Message, "message", "", "Match against the main log message/original access log message")
	flag.StringVar(&conf.UrlBase, "url", conf.UrlBase, "Base URL for the elasticsearch server")
	flag.StringVar(&conf.Username, "username", conf.Username, "Enable basic auth by setting username")
	flag.StringVar(&conf.Password, "password", conf.Password, "Password for basic auth")
	flag.StringVar(&configFile,"config", configFileDef, "Location of TOML formatted configuration file https://github.com/toml-lang/toml\n\tNOTE: setting non-default configfile override comflags\n\t")

}

// Read the configuration file, and return a config struct, using an
// input struct for the default values for each field. Pass any errors
// back up in the second, error argument
func ReadConfig(conf Config, filename string) (Config, error) {
	_, err := toml.DecodeFile( filename, &conf)
	if err != nil {
		return conf, err
	}
	return conf, nil
}

// Perform the query and return a http.Response object
func DoQuery( conf Config ) (*http.Response, error) {
	url := conf.UrlBase
	if conf.Count {
		url += "_count"
	} else {
		url += "_search"
	}
	t, err := template.New("esquery").Parse(reqTemplate)
	if err != nil {
		panic(err)
	}
	var req_body bytes.Buffer
	if debug {
		err = t.Execute( &req_body, conf)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "Configuration: %+v\n",conf)
		fmt.Fprintf(os.Stderr, "URL for query: %s\n",url)
		fmt.Fprintf(os.Stderr, "Query request body:\n%s\n",req_body.String())
	}

	err = t.Execute( &req_body, conf)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("GET", url, &req_body)
	if len(conf.Username) > 0 {
		req.SetBasicAuth( conf.Username, conf.Password)
	}
	
	trans := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: trans}
	return(client.Do(req))
}

func main() {
	flag.Parse()

	if configFile != configFileDef {
		if debug {
			fmt.Fprintf(os.Stderr, "Configuration file: %s\n",configFile)
		}
		conf2, err := ReadConfig( conf, configFile)
		if err == nil {
			conf = conf2
		} else {
			fmt.Fprintf(os.Stderr, "Failed to read file: %s\n",configFile)
			panic(err)
		}
	}

	// Shortcut to assign the first position non-flag argument to the message match
	args := flag.Args()
	if len(args) > 0 {
		searchFilter, err := regexp.Compile("^[A-Za-z0-9_@][A-Za-z0-9_.]*:")
		if err != nil {
			panic( err)
		}
		if len(args) == 1 && ! searchFilter.MatchString(args[0]) {
			conf.Message = args[0]
		} else {
			conf.ExtTerms = args
		}
	}

	// If we have a username but no password, try prompting for it
	if len(conf.Username) > 0 && len(conf.Password) == 0 {
		fmt.Fprintf(os.Stderr, "Please enter password for user %s: ",conf.Username)
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			panic(err)
		}
		conf.Password = string(bytePassword)
		fmt.Fprintf(os.Stderr, "\n")

	}
		
	resp, err := DoQuery(conf)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	// fmt.Println( "response Status :", resp.Status)
	// fmt.Println( "response Headers:", resp.Header)
	body, _ := ioutil.ReadAll( resp.Body)
	fmt.Println(string(body))
}
