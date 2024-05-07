package pgrmongodb

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func httpRequestWithBearerAuth(token string, method string, url string, messagebody string, http_timeout int64) (*http.Response, error) {
	client := &http.Client{Timeout: time.Duration(http_timeout) * time.Second}
	req, err := http.NewRequest(method,
		url,
		bytes.NewBuffer([]byte(messagebody)))
	if err != nil {
		return &http.Response{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	r, err := client.Do(req)
	if err != nil {
		return &http.Response{}, err
	} else {
		return r, nil
	}
}

func responseToArrayOfMap(r *http.Response) ([]map[string]interface{}, error) {
	_, resp, err := responseToMapHelper(r, true)
	return resp, err
}

func responseToMap(r *http.Response) (map[string]interface{}, error) {
	resp, _, err := responseToMapHelper(r, false)
	return resp, err
}

func responseToMapHelper(r *http.Response, isArray bool) (map[string]interface{}, []map[string]interface{}, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, err
	}
	if isArray {
		resp := make([]map[string]interface{}, 1)
		if err := json.Unmarshal(bodyBytes, &resp); err != nil {
			return nil, nil, err
		}
		return nil, resp, nil
	} else {
		resp := make(map[string]interface{})
		if err := json.Unmarshal(bodyBytes, &resp); err != nil {
			return nil, nil, err
		}
		return resp, nil, nil
	}
}

// uses digestParts, getMD5, getCnonce, getDigestAuthrization for digest authentication
// https://stackoverflow.com/a/39481441
func digestRequest(http_method string, uri string, pubkey string, privkey string, http_body []byte, headerAccept string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http_method, uri, bytes.NewBuffer(http_body))
	if err != nil {
		return nil, err
	}
	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// parse 401
	if r.StatusCode != http.StatusUnauthorized {
		return nil, fmt.Errorf("received status code %d but expected 401 for digest authenticated endpoint", r.StatusCode)
	}
	digestParts := digestParts(r)

	digestParts["method"] = http_method
	digestParts["username"] = pubkey
	digestParts["password"] = privkey
	req, err = http.NewRequest(http_method, uri, bytes.NewBuffer(http_body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", getDigestAuthrization(digestParts))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", headerAccept)

	r, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if r.StatusCode >= 200 && r.StatusCode < 300 {
		if http_method == "DELETE" {
			return nil, nil
		} else {
			bodyBytes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			var response map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &response); err != nil { // Parse []byte to go struct pointer
				var altresponse []map[string]interface{}
				if err := json.Unmarshal(bodyBytes, &altresponse); err != nil { // some API response are array of json objects
					return nil, err
				} else {
					response = make(map[string]interface{})
					response["results"] = altresponse
					return response, nil
				}
			}
			return response, nil
		}
	} else {
		return nil, fmt.Errorf("received status code %d but expected 200", r.StatusCode)
	}
}

func digestParts(resp *http.Response) map[string]string {
	result := map[string]string{}
	if len(resp.Header["Www-Authenticate"]) > 0 {
		wantedHeaders := []string{"nonce", "realm", "qop"}
		responseHeaders := strings.Split(resp.Header["Www-Authenticate"][0], ",")
		for _, r := range responseHeaders {
			for _, w := range wantedHeaders {
				if strings.Contains(r, w) {
					result[w] = strings.Split(r, `"`)[1]
				}
			}
		}
	}
	return result
}

func getMD5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getCnonce() string {
	b := make([]byte, 8)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)[:16]
}

func getDigestAuthrization(digestParts map[string]string) string {
	d := digestParts
	ha1 := getMD5(d["username"] + ":" + d["realm"] + ":" + d["password"])
	ha2 := getMD5(d["method"] + ":" + d["uri"])
	nonceCount := 00000013
	cnonce := getCnonce()
	response := getMD5(fmt.Sprintf("%s:%s:%v:%s:%s:%s", ha1, d["nonce"], nonceCount, cnonce, d["qop"], ha2))
	authorization := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", cnonce="%s", nc="%v", qop="%s", response="%s"`,
		d["username"], d["realm"], d["nonce"], d["uri"], cnonce, nonceCount, d["qop"], response)
	return authorization
}
