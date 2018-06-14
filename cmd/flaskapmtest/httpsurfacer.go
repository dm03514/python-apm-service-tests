package main

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

/*
{\"pythonapm.http.request.time_microseconds\": [{\"type\": \"histogram\", \"timestamp\": \"2018-06-05 13:49:12.552794\", \"value\": 675, \"name\": \"pythonapm.http.request.time_microseconds\"}], \"pythonapm.http.request.rss.diff.bytes\": [{\"type\": \"gauge\", \"timestamp\": \"2018-06-05 13:49:12.553209\", \"value\": 0, \"name\": \"pythonapm.http.request.rss.diff.bytes\"}]}","time":"2018-06-05T13:49:12Z"
*/

type metric struct {
	Type      string
	Timestamp string
	Value     float64
	Name      string
}

type response struct {
	Metrics map[string][]metric
}

func assertCorrect(body []byte) error {
	log.WithFields(log.Fields{
		"request": string(body),
	}).Info("asserting_correct")

	var r response
	err := json.Unmarshal(body, &r)
	if err != nil {
		return err
	}

	if _, ok := r.Metrics["pythonapm.http.request.time_microseconds"]; !ok {
		return fmt.Errorf("key %q not found in map", "pythonapm.http.request.time_microseconds")
	}

	if _, ok := r.Metrics["pythonapm.http.request.rss.diff.bytes"]; !ok {
		return fmt.Errorf("key %q not found in map", "pythonapm.http.request.rss.diff.bytes")
	}

	if _, ok := r.Metrics["pythonapm.instruments.allocators.str.count"]; !ok {
		return fmt.Errorf("key %q not found in map", "pythonapm.instruments.allocators.str.count")
	}

	return nil
}

type handler struct {
	requestsChan chan []byte
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()
	h.requestsChan <- bs
	http.Error(w, "OK", http.StatusOK)
}

// testHTTPSurfacer starts an HTTP server, exercises flask APM, then waits
// and asserts on the output
func testHTTPSurfacer(addr string, path string, testServerAddr string, timeout time.Duration) error {
	requestsChan := make(chan []byte, 1)
	timer := time.NewTimer(timeout)

	var wg sync.WaitGroup
	wg.Add(1)

	h := &http.Server{
		Addr: testServerAddr,
		Handler: handler{
			requestsChan: requestsChan,
		},
	}

	go func() {
		log.WithFields(log.Fields{
			"addr": testServerAddr,
		}).Info("starting_http_listener")

		if err := h.ListenAndServe(); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("metric_server_listen_and_serve")
		}
	}()

	// apply the initial request input
	// just focused on a test skeleton, in a company project would
	// handle errors from here instead of panic'ing
	go func() {
		log.WithFields(log.Fields{
			"addr": addr,
			"path": path,
		}).Info("making_flask_request")

		resp, err := http.Get("http://" + addr + path)

		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		// verify that response is correct, in real script would handle errors gracefully
		// instead of panic'ing
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			panic(fmt.Errorf("Flask request failed: %+v", resp))
		}

		header := resp.Header.Get("dm03514/pythonapm")
		if header == "" {
			panic(fmt.Errorf("expected header %q to have a value, found %q", "dm03514/pythonapm", header))
		}

		wg.Done()
	}()

	log.WithFields(log.Fields{
		"timeout": timeout,
	}).Info("http_test_starting")

forloop:
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout %s reached", timeout)
		case r := <-requestsChan:
			if err := assertCorrect(r); err != nil {
				return err
			}
			break forloop
		}
	}

	wg.Wait()

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	h.Shutdown(ctx)

	// ignores graceful server cleanup
	log.WithFields(log.Fields{
		"status": "PASS",
	}).Info("test_complete")

	return nil
}
