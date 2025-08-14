package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Evaluator struct {
	base string
	cl   *http.Client
}

func NewEvaluator(base string, timeout time.Duration) *Evaluator {
	if base == "" { base = "http://prometheus:9090" }
	return &Evaluator{base: base, cl: &http.Client{Timeout: timeout}}
}

func (e *Evaluator) QueryRange(ctx context.Context, query, window string) (float64, error) {
	u, _ := url.Parse(e.base + "/api/v1/query")
	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	resp, err := e.cl.Do(req); if err != nil { return 0, err }
	defer resp.Body.Close()
	var payload struct {
		Status string `json:"status"`
		Data struct {
			ResultType string `json:"resultType"`
			Result []struct{
				Value [2]any `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil { return 0, err }
	if len(payload.Data.Result) == 0 { return 0, nil }
	// value -> [timestamp, "value"]
	valStr, _ := payload.Data.Result[0].Value[1].(string)
	var f float64
	fmt.Sscanf(valStr, "%f", &f)
	return f, nil
}