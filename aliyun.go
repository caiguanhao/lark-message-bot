package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// aliyunClient is a slim client for the Aliyun DNS (alidns) RPC API,
// see https://help.aliyun.com/document_detail/29772.html
type aliyunClient struct {
	AccessKeyId     string
	AccessKeySecret string

	MainDomain string // main domain to create subdomains under, e.g. example.com
	DefaultIP  string // default IP address of the new A records
}

type aliyunAPIError struct {
	Code      string
	Message   string
	RequestId string
}

func (e aliyunAPIError) Error() string {
	return fmt.Sprintf("aliyun %s: %s", e.Code, e.Message)
}

func (c *aliyunClient) configured() bool {
	return c != nil && c.AccessKeyId != "" && c.AccessKeySecret != "" &&
		c.MainDomain != "" && c.DefaultIP != ""
}

// AddARecord adds an A record rr.MainDomain -> value and returns its record id.
func (c *aliyunClient) AddARecord(rr, value string) (string, error) {
	var result struct {
		RecordId string
	}
	err := c.do("AddDomainRecord", map[string]string{
		"DomainName": c.MainDomain,
		"RR":         rr,
		"Type":       "A",
		"Value":      value,
	}, &result)
	if err != nil {
		return "", err
	}
	return result.RecordId, nil
}

type DomainRecord struct {
	RecordId        string
	RR              string
	Type            string
	Value           string
	CreateTimestamp int64 // in milliseconds
}

// ListARecords lists all A records of the main domain, newest first.
func (c *aliyunClient) ListARecords() ([]DomainRecord, error) {
	var records []DomainRecord
	for page := 1; ; page++ {
		var result struct {
			TotalCount    int
			DomainRecords struct {
				Record []DomainRecord
			}
		}
		err := c.do("DescribeDomainRecords", map[string]string{
			"DomainName": c.MainDomain,
			"Type":       "A",
			"PageNumber": strconv.Itoa(page),
			"PageSize":   "500", // max page size allowed by the api
		}, &result)
		if err != nil {
			return nil, err
		}
		records = append(records, result.DomainRecords.Record...)
		if len(records) >= result.TotalCount || len(result.DomainRecords.Record) == 0 {
			break
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreateTimestamp > records[j].CreateTimestamp
	})
	return records, nil
}

func (c *aliyunClient) do(action string, params map[string]string, result interface{}) error {
	for k, v := range map[string]string{
		"Format":           "JSON",
		"Version":          "2015-01-09",
		"AccessKeyId":      c.AccessKeyId,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   fmt.Sprintf("%d", time.Now().UnixNano()),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Action":           action,
	} {
		params[k] = v
	}
	params["Signature"] = c.sign(params)

	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://alidns.aliyuncs.com/?" + values.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr aliyunAPIError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Code != "" {
			return apiErr
		}
		return fmt.Errorf("aliyun error (%s): %s", resp.Status, body)
	}
	return json.Unmarshal(body, result)
}

// sign calculates the HMAC-SHA1 signature of the parameters,
// see https://help.aliyun.com/document_detail/315526.html
func (c *aliyunClient) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	query := make([]string, 0, len(keys))
	for _, k := range keys {
		query = append(query, aliyunPercentEncode(k)+"="+aliyunPercentEncode(params[k]))
	}
	stringToSign := "GET&%2F&" + aliyunPercentEncode(strings.Join(query, "&"))
	mac := hmac.New(sha1.New, []byte(c.AccessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// aliyunPercentEncode encodes s the way the Aliyun signature requires (RFC 3986).
func aliyunPercentEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}
