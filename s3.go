package awsauth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func signatureS3(stringToSign string, keys Credentials) string {
	hash := hmac.New(sha1.New, []byte(keys.SecretAccessKey))
	hash.Write([]byte(stringToSign))
	signature := make([]byte, base64.StdEncoding.EncodedLen(hash.Size()))
	base64.StdEncoding.Encode(signature, hash.Sum(nil))
	return string(signature)
}

func stringToSignS3(request *http.Request) string {
	// This is a hack to work for presigning urls for browsers that don't set the date header
	str := request.Method + "\n"
	str += request.Header.Get("Content-Md5") + "\n"
	str += request.Header.Get("Content-Type") + "\n"

	if request.Header.Get("x-amz-date") != "" {
		str += "\n"
	} else {
		str += request.Header.Get("Date") + "\n"
	}

	canonicalHeaders := canonicalAmzHeadersS3(request)
	if canonicalHeaders != "" {
		str += canonicalHeaders
	}

	str += canonicalResourceS3(request)

	return str
}

func stringToSignS3Url(method string, expire time.Time, path string) string {
	return method + "\n\n\n" + timeToUnixEpochString(expire) + "\n" + path
}

func timeToUnixEpochString(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

func canonicalAmzHeadersS3(request *http.Request) string {
	var headers []string

	for header := range request.Header {
		standardized := strings.ToLower(strings.TrimSpace(header))
		if strings.HasPrefix(standardized, "x-amz") {
			headers = append(headers, standardized)
		}
	}

	sort.Strings(headers)

	for i, header := range headers {
		headers[i] = header + ":" + strings.Replace(request.Header.Get(header), "\n", " ", -1)
	}

	if len(headers) > 0 {
		return strings.Join(headers, "\n") + "\n"
	} else {
		return ""
	}
}

func canonicalResourceS3(request *http.Request) string {
	res := ""

	if isS3VirtualHostedStyle(request) {
		bucketname := strings.Split(request.Host, ".")[0]
		res += "/" + bucketname
	}

	res += normuri(request.URL.Path)
	//TODO: This is a hack
	if request.URL.RawQuery != "" {
		res += "?" + request.URL.RawQuery
	}

	// for _, subres := range strings.Split(subresourcesS3, ",") {
	// 	if strings.HasPrefix(request.URL.RawQuery, subres) {
	// 		res += "?" + subres
	// 	}
	// }

	return res
}

func prepareRequestS3(request *http.Request) *http.Request {
	request.Header.Set("Date", timestampS3())
	if request.URL.Path == "" {
		request.URL.Path += "/"
	}
	return request
}

// Info: http://docs.aws.amazon.com/AmazonS3/latest/dev/VirtualHosting.html
func isS3VirtualHostedStyle(request *http.Request) bool {
	service, _ := serviceAndRegion(request.Host)
	return service == "s3" && strings.Count(request.Host, ".") == 3
}

func timestampS3() string {
	return now().Format(timeFormatS3)
}

const (
	timeFormatS3   = time.RFC1123Z
	subresourcesS3 = "acl,lifecycle,location,logging,notification,partNumber,policy,requestPayment,torrent,uploadId,uploads,versionId,versioning,versions,website"
)
