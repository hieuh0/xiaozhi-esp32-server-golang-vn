package mcp_market

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func BuildHeaders(auth AuthConfig) map[string]string {
	headers := make(map[string]string)

	for k, v := range auth.ExtraHeaders {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		headers[k] = v
	}

	token := strings.TrimSpace(auth.Token)
	if token == "" {
		return headers
	}

	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case AuthTypeHeader:
		headerName := strings.TrimSpace(auth.HeaderName)
		if headerName == "" {
			headerName = "Authorization"
		}
		headers[headerName] = token
	case AuthTypeBearer:
		fallthrough
	default:
		headers["Authorization"] = "Bearer " + token
	}

	return headers
}

func BuildCookies(auth AuthConfig, providerID string) map[string]string {
	token := strings.TrimSpace(auth.Token)
	if token == "" {
		return nil
	}

	switch NormalizeProviderID(providerID) {
	case ProviderModelScope:
		return map[string]string{"m_session_id": token}
	default:
		return nil
	}
}

func FetchJSON(ctx context.Context, endpoint string, headers map[string]string, opts HTTPOptions) (interface{}, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}

	method := strings.ToUpper(strings.TrimSpace(opts.Method))
	if method == "" {
		method = http.MethodGet
	}

	var bodyReader io.Reader
	if opts.JSONBody != nil {
		bodyBytes, err := json.Marshal(opts.JSONBody)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	endpointURL := strings.TrimSpace(endpoint)
	if len(opts.Query) > 0 {
		u, err := url.Parse(endpointURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse request URL: %w", err)
		}
		q := u.Query()
		for k, v := range opts.Query {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		endpointURL = u.String()
	}

	client := &http.Client{Timeout: opts.Timeout}
	req, err := http.NewRequestWithContext(ctx, method, endpointURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if opts.JSONBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for k, v := range opts.Cookies {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		req.AddCookie(&http.Cookie{Name: k, Value: v, Path: "/"})
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("request failed (%d): %s", resp.StatusCode, msg)
	}

	if len(body) == 0 {
		return map[string]interface{}{}, nil
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return data, nil
}

func BuildDetailURL(catalogURL, detailURLTemplate, serviceID string) (string, error) {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		return "", fmt.Errorf("service_id must not be empty")
	}

	template := strings.TrimSpace(detailURLTemplate)
	if template != "" {
		if strings.Contains(template, "{raw_id}") {
			template = strings.ReplaceAll(template, "{raw_id}", serviceID)
		}
		return strings.ReplaceAll(template, "{id}", url.PathEscape(serviceID)), nil
	}

	base, err := url.Parse(strings.TrimSpace(catalogURL))
	if err != nil {
		return "", fmt.Errorf("catalog_url is invalid: %w", err)
	}

	base.Path = path.Join(base.Path, url.PathEscape(serviceID))
	return base.String(), nil
}

func ExtractServiceList(payload interface{}) []map[string]interface{} {
	items := findFirstObjectArray(payload)
	if len(items) > 0 {
		return items
	}
	return []map[string]interface{}{}
}

func ParseServiceSummary(item map[string]interface{}) (id, name, description string) {
	id = firstString(item,
		"service_id", "id", "slug", "name", "serviceName", "serviceId", "tool_id", "toolId",
	)
	name = firstString(item,
		"name", "title", "service_name", "serviceName", "tool_name", "toolName", "id",
	)
	description = firstString(item,
		"description", "desc", "summary", "intro", "detail",
	)
	if id == "" {
		id = name
	}
	if name == "" {
		name = id
	}
	return id, name, description
}

func ParseServiceDetail(payload interface{}, marketID uint, marketName, serviceID string, headers map[string]string) (*MarketServiceDetail, error) {
	rawMap := asMap(payload)
	if rawMap == nil {
		rawMap = map[string]interface{}{"data": payload}
	}

	name := firstString(rawMap, "name", "title", "service_name", "serviceName")
	desc := firstString(rawMap, "description", "desc", "summary", "intro")
	if name == "" {
		if node := findByKeys(rawMap, "name", "title", "service_name", "serviceName"); node != nil {
			name = asString(node)
		}
	}
	if desc == "" {
		if node := findByKeys(rawMap, "description", "desc", "summary", "intro"); node != nil {
			desc = asString(node)
		}
	}
	if name == "" {
		name = serviceID
	}

	endpoints := ExtractMCPEndpoints(payload, headers)

	return &MarketServiceDetail{
		MarketID:    marketID,
		MarketName:  marketName,
		ServiceID:   serviceID,
		Name:        name,
		Description: desc,
		Endpoints:   endpoints,
		Raw:         rawMap,
	}, nil
}

func ExtractMCPEndpoints(payload interface{}, headers map[string]string) []ParsedEndpoint {
	ret := make([]ParsedEndpoint, 0)
	seen := make(map[string]struct{})
	var mu sync.Mutex

	appendEndpoint := func(ep ParsedEndpoint) {
		ep.Transport = normalizeTransport(ep.Transport)
		ep.URL = strings.TrimSpace(ep.URL)
		if ep.URL == "" {
			return
		}
		if ep.Transport != TransportSSE && ep.Transport != TransportStreamableHTTP {
			return
		}
		key := ep.Transport + "|" + NormalizeURL(ep.URL)
		mu.Lock()
		defer mu.Unlock()
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		if len(headers) > 0 {
			ep.Headers = cloneHeaders(headers)
		}
		ret = append(ret, ep)
	}

	walkAny(payload, func(node interface{}) {
		m := asMap(node)
		if m == nil {
			return
		}

		if configNode, ok := m["mcpServers"]; ok {
			if smap := asMap(configNode); smap != nil {
				for name, v := range smap {
					if cfg := asMap(v); cfg != nil {
						transportType := firstString(cfg, "type", "transport", "protocol", "transport_type")
						endpoint := firstString(cfg, "url", "endpoint", "sse_url", "sseUrl")
						if endpoint != "" {
							appendEndpoint(ParsedEndpoint{Name: name, Transport: transportType, URL: endpoint})
						}
					}
				}
			}
		}

		if operationalNode, ok := m["operational_urls"]; ok {
			appendEndpointFromArray(operationalNode, appendEndpoint)
		}
		if mcpServersNode, ok := m["mcp_servers"]; ok {
			appendEndpointFromArray(mcpServersNode, appendEndpoint)
		}

		transportType := firstString(m, "type", "transport", "protocol", "transport_type")
		if urlValue := firstString(m, "url", "endpoint", "mcp_url", "mcpUrl"); urlValue != "" {
			if transportType == "" {
				transportType = inferTransportFromURL(urlValue)
			}
			appendEndpoint(ParsedEndpoint{Name: firstString(m, "name", "title"), Transport: transportType, URL: urlValue})
		}
		if sseURL := firstString(m, "sse_url", "sseUrl", "sse"); sseURL != "" {
			appendEndpoint(ParsedEndpoint{Name: firstString(m, "name", "title"), Transport: TransportSSE, URL: sseURL})
		}
		if shURL := firstString(m, "streamablehttp", "streamable_http", "http"); shURL != "" {
			appendEndpoint(ParsedEndpoint{Name: firstString(m, "name", "title"), Transport: TransportStreamableHTTP, URL: shURL})
		}
	})

	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Transport == ret[j].Transport {
			return ret[i].URL < ret[j].URL
		}
		return ret[i].Transport < ret[j].Transport
	})

	return ret
}

func NormalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return strings.TrimSuffix(strings.ToLower(raw), "/")
	}
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimSuffix(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}

	query := u.Query()
	if len(query) > 0 {
		keys := make([]string, 0, len(query))
		for k := range query {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ordered := url.Values{}
		for _, k := range keys {
			vals := query[k]
			sort.Strings(vals)
			for _, v := range vals {
				ordered.Add(k, v)
			}
		}
		u.RawQuery = ordered.Encode()
	}

	return u.String()
}

func normalizeTransport(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "sse":
		return TransportSSE
	case "streamablehttp", "streamable_http", "http", "streamable-http":
		return TransportStreamableHTTP
	default:
		return v
	}
}

func cloneHeaders(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func appendEndpointFromArray(v interface{}, appendFn func(ParsedEndpoint)) {
	items := asArray(v)
	if len(items) == 0 {
		return
	}
	for _, item := range items {
		m := asMap(item)
		if m == nil {
			continue
		}
		rawURL := firstString(m, "url", "endpoint")
		if rawURL == "" {
			continue
		}
		transportType := firstString(m, "type", "transport", "protocol", "transport_type")
		if transportType == "" {
			transportType = inferTransportFromURL(rawURL)
		}
		appendFn(ParsedEndpoint{Name: firstString(m, "name", "title"), Transport: transportType, URL: rawURL})
	}
}

func inferTransportFromURL(rawURL string) string {
	lowerURL := strings.ToLower(strings.TrimSpace(rawURL))
	switch {
	case strings.HasSuffix(lowerURL, "/sse"), strings.Contains(lowerURL, "/sse?"):
		return TransportSSE
	case strings.HasSuffix(lowerURL, "/streamable_http"), strings.Contains(lowerURL, "/streamable_http?"):
		return TransportStreamableHTTP
	case strings.HasSuffix(lowerURL, "/streamablehttp"), strings.Contains(lowerURL, "/streamablehttp?"):
		return TransportStreamableHTTP
	default:
		return ""
	}
}

func findFirstObjectArray(v interface{}) []map[string]interface{} {
	switch t := v.(type) {
	case []interface{}:
		return convertObjectArray(t)
	case map[string]interface{}:
		for _, key := range []string{"data", "items", "services", "list", "results", "records", "mcp_server_list"} {
			if next, ok := t[key]; ok {
				if arr := findFirstObjectArray(next); len(arr) > 0 {
					return arr
				}
			}
		}
		for _, val := range t {
			if arr := findFirstObjectArray(val); len(arr) > 0 {
				return arr
			}
		}
	}
	return nil
}

func convertObjectArray(arr []interface{}) []map[string]interface{} {
	ret := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		if m := asMap(item); m != nil {
			ret = append(ret, m)
		}
	}
	return ret
}

func walkAny(v interface{}, visit func(interface{})) {
	visit(v)
	switch t := v.(type) {
	case map[string]interface{}:
		for _, child := range t {
			walkAny(child, visit)
		}
	case []interface{}:
		for _, child := range t {
			walkAny(child, visit)
		}
	}
}

func findByKeys(v interface{}, keys ...string) interface{} {
	target := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		target[k] = struct{}{}
	}
	var found interface{}
	walkAny(v, func(node interface{}) {
		if found != nil {
			return
		}
		if m := asMap(node); m != nil {
			for k, val := range m {
				if _, ok := target[k]; ok {
					found = val
					return
				}
			}
		}
	})
	return found
}

func asMap(v interface{}) map[string]interface{} {
	m, ok := v.(map[string]interface{})
	if ok {
		return m
	}
	return nil
}

func asArray(v interface{}) []interface{} {
	if arr, ok := v.([]interface{}); ok {
		return arr
	}
	return nil
}

func asString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func firstString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			if s := asString(val); s != "" {
				return s
			}
		}
	}
	return ""
}
