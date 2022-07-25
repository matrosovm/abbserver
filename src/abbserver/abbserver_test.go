package abbserver

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGet(t *testing.T) {
	tt := []struct {
		name       string
		input      string
		unique 	   bool
		statusCode int
	}{
		{
			name:       "First url",
			input:      "www.google.com",
			unique: 	true,
			statusCode: http.StatusOK,
		},
		{
			name:       "Second unique url",
			input:      "yandex.ru",
			unique: 	true,
			statusCode: http.StatusOK,
		},
		{
			name:       "Third unique url",
			input:      "mail.ru",
			unique: 	true,
			statusCode: http.StatusOK,
		},
		{
			name:       "Fourth unique url",
			input:      "job.ozon.ru/internships/",
			unique: 	true,
			statusCode: http.StatusOK,
		},
		{
			name:       "Repeat",
			input:      "www.google.com",
			unique: 	false,
			statusCode: http.StatusOK,
		},
	}

	
	db = &localDB{LinkURL: make(map[link]string), 
		URLLink: map[string]link{}}
	
	links := make(map[link]struct{})
	
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", 
				bytes.NewBuffer([]byte(tc.input)))
			w := httptest.NewRecorder()

			handler(w, request)

			if w.Code != tc.statusCode {
				t.Errorf("Want status '%d', got '%d'", tc.statusCode, w.Code)
			}
			res := w.Result()
			defer res.Body.Close()

			data, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Errorf("Expected nil error, got %v", err)
			}
			if len(data) < 10 {
				t.Errorf("Message is shorter than link")
			}
			tmp := bytesToLink(data[len(data) - 11:])
			if _, ok := links[tmp]; ok == tc.unique {
				t.Errorf("Expected %t new value, got %t new value", 
				tc.unique, ok)
			}
			links[tmp] = struct{}{}
		})
	}
}

func TestPost(t *testing.T) {
	urls := []string{"www.google.com", "yandex.ru", "mail.ru", 
		"job.ozon.ru/internships/"}
	
	db = &localDB{LinkURL: make(map[link]string), 
		URLLink: map[string]link{}}
	links := make([]link, len(urls) + 1)
	for i, url := range urls {
		request := httptest.NewRequest(http.MethodPost, "/", 
			bytes.NewBuffer([]byte(url)))
		w := httptest.NewRecorder()
		handler(w, request)
		
		res := w.Result()
		defer res.Body.Close()

		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if len(data) < 10 {
			t.Errorf("Message is shorter than link")
		}
		tmp := bytesToLink(data[len(data) - 11:])
		links[i] = tmp
	}
	links[len(urls)] = generateNextUniqLink()
	
	tt := []struct {
		name       string
		input      string
		inDB 	   bool
		statusCode int
	}{
		{
			name:       "First url in db",
			input:      "www.google.com",
			inDB:       true,
			statusCode: http.StatusOK,
		},
		{
			name:       "Second url in db",
			input:      "yandex.ru",
			inDB:       true,
			statusCode: http.StatusOK,
		},
		{
			name:       "Third url in db",
			input:      "mail.ru",
			inDB:       true,
			statusCode: http.StatusOK,
		},
		{
			name:       "Fourth url in db",
			input:      "job.ozon.ru/internships/",
			inDB:       true,
			statusCode: http.StatusOK,
		},
		{
			name:       "Not added to the database",
			input:      "job.ozon.ru",
			inDB:       false,
			statusCode: http.StatusNotFound,
		},
	}
	
	for i, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", 
				bytes.NewBuffer([]byte(string(links[i][:]))))
			w := httptest.NewRecorder()

			handler(w, request)

			if w.Code != tc.statusCode {
				t.Errorf("Want status '%d', got '%d'", 
					tc.statusCode, w.Code)
			}
			res := w.Result()
			defer res.Body.Close()

			data, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Errorf("Expected nil error, got %v", err)
			}
			if len(data) < 10 {
				t.Errorf("Message is shorter than link")
			}
			if !tc.inDB {
				if str := string(data); str != "URL not found\n" {
					t.Errorf("Expected url not found, got %s", str)
				}
			} else {
				url := string(data[27:])
				if tc.input + "\n" != url {
					t.Errorf("Expected %s new value, got %s new value", 
						tc.input, url)
				}
			}
		})
	}
}