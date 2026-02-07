package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

func GET[T any](ctx context.Context, client *http.Client, api string, header map[string]string) (T, int, error) {
	var result T

	if client == nil {
		client = &http.Client{}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", api, nil)
	if err != nil {
		return result, 0, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, 0, err
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, statusCode, err
	}
	return result, statusCode, nil
}

func POSTForm[T any](ctx context.Context, client *http.Client, api string, header map[string]string, form url.Values) (T, int, error) {
	var result T

	if client == nil {
		client = &http.Client{}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", api, strings.NewReader(form.Encode()))
	if err != nil {
		return result, 0, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	for k, v := range header {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, 0, err
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, statusCode, err
	}
	return result, statusCode, nil
}

func POSTJson[T any](ctx context.Context, client *http.Client, api string, header map[string]string, body any) (T, int, error) {
	var result T

	if client == nil {
		client = &http.Client{}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return result, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", api, bytes.NewReader(jsonBody))
	if err != nil {
		return result, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for k, v := range header {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, 0, err
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, statusCode, err
	}
	return result, statusCode, nil
}
