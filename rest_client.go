package GoRest
import (
    "net/http"
    "fmt"
    "strings"
    u "net/url"
    "io/ioutil"
    "bytes"
    "errors"
)

type RestClient struct {
    client      *http.Client
    url         string
    accept      MediaType
    contentType MediaType
    headers     map[string]string
    query       map[string]string
    cookies     []*http.Cookie
}

// The constructor for an immutable RestClient
//
// The baseUrl argument should be a fully qualified url and will be validated on one of the HTTP methods:
// https://github.com/ | https://groups.google.com/forum/#!forum/golang-nuts
//
// By providing the baseUrl to the RestClient it can be stored in a partial state to be built upon
// in a route or handler function
func MakeClient(baseUrl string) RestClient {
    return newClient(
            &http.Client{},
            strings.Trim(baseUrl, "/"),
            ApplicationJSON,
            ApplicationJSON,
            make(map[string]string),
            make(map[string]string),
            nil)
}

// Private constructor used to provide all RestClient parameters
func newClient(client *http.Client, url string, accept MediaType, contentType MediaType, headers map[string]string,
query map[string]string, cookies []*http.Cookie) RestClient {
    return RestClient{
        client:         client,
        url:            url,
        accept:         accept,
        contentType:    contentType,
        headers:        headers,
        query:          query,
        cookies:        cookies}
}

// ===================================================================
//                        RestClient Getters
// ===================================================================

func (rc RestClient) GetURL() string {
    return rc.url
}

func (rc RestClient) GetAccept() MediaType {
    return rc.accept
}

func (rc RestClient) GetContentType() MediaType {
    return rc.contentType
}

func (rc RestClient) GetHeaders() map[string]string {
    return rc.headers
}

// ===================================================================
//                     Immutable Builder Methods
// ===================================================================

func (rc RestClient) Accept(accept MediaType) RestClient {
    return newClient(rc.client, rc.url, accept, rc.contentType, rc.headers, rc.query, rc.cookies)
}

func (rc RestClient) ContentType(contentType MediaType) RestClient {
    return newClient(rc.client, rc.url, rc.accept, contentType, rc.headers, rc.query, rc.cookies)
}

func (rc RestClient) Path(path ...string) RestClient {
    newClient := newClient(rc.client, rc.url, rc.accept, rc.contentType, rc.headers, rc.query, rc.cookies)
    for _, p := range path { newClient.url = fmt.Sprintf("%s/%s", newClient.url, strings.Trim(p, "/")) }
    return newClient
}

func (rc RestClient) Query(key, value string) RestClient {
    newQuery := make(map[string]string)
    for k, v := range rc.headers { newQuery[k] = v }
    newQuery[key] = value
    return newClient(rc.client, rc.url, rc.accept, rc.contentType, rc.headers, newQuery, rc.cookies)
}

func (rc RestClient) Header(key, value string) RestClient {
    newHeaders := make(map[string]string)
    for k, v := range rc.headers { newHeaders[k] = v }
    newHeaders[key] = value
    return newClient(rc.client, rc.url, rc.accept, rc.contentType, newHeaders, rc.query, rc.cookies)
}

func (rc RestClient) Cookie(cookie *http.Cookie) RestClient {
    return newClient(rc.client, rc.url, rc.accept, rc.contentType, rc.headers, rc.query, append(rc.cookies, cookie))
}

func (rc RestClient) Get(resEntity ...interface{}) error {
    return rc.request("GET", nil, resEntity...)
}

func (rc RestClient) Put(reqBody []byte, resEntity ...interface{}) error {
    return rc.request("PUT", reqBody, resEntity...)
}

func (rc RestClient) Post(reqBody []byte, resEntity ...interface{}) error {
    return rc.request("POST", reqBody, resEntity...)
}

func (rc RestClient) Delete(entity ...interface{}) error {
    return nil
}

// The main request function. This handles building out the request and reading the response into
// the provided resEntity
func (rc RestClient) request(httpReq string, reqBody []byte, resEntity ...interface{}) error {
    // Validate the URL
    uri, err := u.Parse(rc.url);
    if err != nil { return err }

    // Add query params
    for k, v := range rc.query { uri.Query().Add(k, v) }

    var req *http.Request

    // Build the Request
    if reqBody != nil {
        req, err = http.NewRequest(httpReq, uri.String(), bytes.NewBuffer(reqBody))
    } else {
        req, err = http.NewRequest(httpReq, uri.String(), nil)
    }
    if err != nil { return err }

    // Add headers
    req.Header.Add("Accept", rc.accept.String())
    req.Header.Add("Content-Type", rc.contentType.String())

    // Make Request
    res, err := rc.client.Do(req)
    if err != nil { return err }

    // Validate the response content type matches the accept type.
    // This is required to allow unmarshalling to the resEntity
    if contentType := res.Header.Get("Content-Type"); len(resEntity) != 0 &&
    !strings.Contains(strings.ToLower(contentType), strings.ToLower(rc.accept.String())) {
        return errors.New(fmt.Sprintf("Expected Response Content-Type [%s] to match/contain Request Accept [%s]",
        contentType, rc.accept.String()))
    }

    body, err := ioutil.ReadAll(res.Body)
    if err != nil { return err }

    // If entities were passed in then unmarshal the body into each
    for _, e := range resEntity {
        if err = rc.accept.Unmarshal(body, e); err != nil { return err }
    }

    // Return success
    return nil
}