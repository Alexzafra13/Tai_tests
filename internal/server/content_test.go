package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func doJSON(t *testing.T, c *http.Client, method, url string, body any, out any) int {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, r)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil && err != io.EOF {
			t.Fatalf("decode %s %s: %v", method, url, err)
		}
	}
	return resp.StatusCode
}

func TestContentAPIRequiresAuth(t *testing.T) {
	ts, c := newTestServer(t)
	for _, path := range []string{"/api/syllabus", "/api/sources", "/api/questions"} {
		if resp, _ := get(t, c, ts.URL+path); resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s without session: %d, want 401", path, resp.StatusCode)
		}
	}
}

func TestQuestionAPIValidation(t *testing.T) {
	ts, c := newTestServer(t)
	login(t, c, ts.URL)

	var created struct{ ID int64 }
	status := doJSON(t, c, "POST", ts.URL+"/api/sources", map[string]any{
		"kind": "law", "title": "Ley 39/2015", "reference": "BOE-A-2015-10565",
		"full_text": "Artículo 21. 2. Este plazo no podrá exceder de seis meses salvo que una norma con rango de Ley establezca uno mayor.",
	}, &created)
	if status != http.StatusCreated || created.ID == 0 {
		t.Fatalf("create source: %d %+v", status, created)
	}

	question := map[string]any{
		"stem":         "¿Plazo máximo general para notificar la resolución expresa?",
		"options":      []string{"Tres meses", "Seis meses", "Un año", "Dos meses"},
		"correct":      1,
		"origin":       "law",
		"author":       "manual",
		"source_id":    created.ID,
		"source_ref":   "art. 21.2",
		"source_quote": "Este plazo no podrá exceder de tres meses", // not in the text
		"status":       "draft",
	}
	var verr struct {
		Error  string
		Fields map[string]string
	}
	if status := doJSON(t, c, "POST", ts.URL+"/api/questions", question, &verr); status != http.StatusUnprocessableEntity {
		t.Fatalf("invented quote: %d, want 422", status)
	}
	if verr.Fields["source_quote"] == "" {
		t.Fatalf("expected source_quote field error, got %+v", verr)
	}

	var check struct{ Found bool }
	doJSON(t, c, "POST", ts.URL+"/api/sources/1/check-quote", map[string]string{"quote": "no podrá exceder de seis meses"}, &check)
	if !check.Found {
		t.Error("check-quote: literal fragment not found")
	}

	question["source_quote"] = "Este plazo no podrá exceder de seis meses"
	var q struct{ ID int64 }
	if status := doJSON(t, c, "POST", ts.URL+"/api/questions", question, &q); status != http.StatusCreated {
		t.Fatalf("valid question: %d", status)
	}

	var page struct {
		Total int
		Items []struct {
			ID          int64
			SourceTitle string `json:"source_title"`
		}
	}
	doJSON(t, c, "GET", ts.URL+"/api/questions?status=draft", nil, &page)
	if page.Total != 1 || page.Items[0].SourceTitle != "Ley 39/2015" {
		t.Fatalf("list: %+v", page)
	}

	if status := doJSON(t, c, "DELETE", ts.URL+"/api/sources/1", nil, nil); status != http.StatusConflict {
		t.Fatalf("delete used source: %d, want 409", status)
	}
}
