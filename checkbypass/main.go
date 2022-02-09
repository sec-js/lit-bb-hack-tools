package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"

func main() {
	if runtime.GOOS == "windows" {
		Reset = ""
		Red = ""
		Green = ""
	}
	helpPtr := flag.Bool("h", false, "Show usage.")
	payloadPtr := flag.String("p", "", "Input payload.")
	flag.Parse()
	if *helpPtr {
		help()
	}
	if *payloadPtr != "" {
		TestWAF(*payloadPtr)
	} else {
		fmt.Println("Payload required.")
		os.Exit(1)
	}
}

//help shows the usage
func help() {
	var usage = `Take as input on stdin a payload and print on stdout all the successful WAF bypasses.
	$> checkbypass -p "<script>alert()</script>"`
	fmt.Println()
	fmt.Println(usage)
	fmt.Println()
	os.Exit(0)
}

//RemoveDuplicateValues >
func RemoveDuplicateValues(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

//ReplaceParameters >
func ReplaceParameters(input string, payload string) string {
	u, err := url.Parse(input)
	if err != nil {
		return ""
	}
	decodedValue, err := url.QueryUnescape(u.RawQuery)
	if err != nil {
		return ""
	}
	var queryResult = ""
	couples := strings.Split(decodedValue, "&")
	for _, pair := range couples {
		values := strings.Split(pair, "=")
		queryResult += values[0] + "=" + url.QueryEscape(payload) + "&"
	}
	return u.Scheme + "://" + u.Host + u.Path + "?" + queryResult[:len(queryResult)-1]
}

//GetRequest performs a GET request
func GetRequest(target string) (string, int, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 20,
	}
	resp, err := netClient.Get(target)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	//Convert the body to type string
	sb := string(body)
	return sb, len(sb), nil
}

type WAF struct {
	Name          string
	Url           string
	BlockedStatus int
	BlockedString string
}

var wafs = []WAF{
	{"Cloudflare",
		"https://www.cloudflare.com/",
		403,
		"you have been blocked"},
	{"Akamai",
		"https://www.akamai.com/",
		403,
		"You don't have permission to access"},
	{"F5",
		"https://www.f5.com/",
		403,
		"The requested URL was rejected. Please consult with your administrator."},
	{"CloudFront",
		"https://docs.aws.amazon.com/",
		403,
		"Generated by cloudfront (CloudFront)"},
	{"Fortiweb",
		"https://www.fortinet.com/",
		403,
		"Web Page Blocked"},
	{"Imperva",
		"https://www.imperva.com/",
		403,
		"Request unsuccessful. Incapsula incident ID"},
	{"Wordfence",
		"https://www.wordfence.com/products/",
		403,
		"Your access to this service has been limited. (HTTP response code 403)"},
}

//TestWAF >
func TestWAF(payload string) {
	var distance = 12
	for _, elem := range wafs {
		url := ReplaceParameters(elem.Url, "test="+payload)
		resp, status, err := GetRequest(url)
		if err != nil {
			fmt.Println(Red + "[ ERROR:-( ] " + Reset + err.Error())
			continue
		}
		if strings.Contains(resp, elem.BlockedString) {
			fmt.Println(Red + "[ BLOCKED! ] " + Reset + elem.Name + strings.Repeat(" ", distance-len(elem.Name)) + " : " + url)
			continue
		}
		if status == elem.BlockedStatus {
			fmt.Println(Red + "[ BLOCKED! ] " + Reset + elem.Name + strings.Repeat(" ", distance-len(elem.Name)) + " : " + url)
			continue
		}
		fmt.Println(Green + "[ BYPASSED ] " + Reset + elem.Name + strings.Repeat(" ", distance-len(elem.Name)) + " : " + url)
	}
}