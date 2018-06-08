package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// waitHTTPready waits until a server is read
// if the server is not ready within the timeout than
// error is returned
func waitHTTPReady(addr string, path string, timeout time.Duration) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	log.WithFields(log.Fields{
		"addr": addr,
		"path": path,
	}).Info("wait_http_ready")

	for {
		select {
		case <-ticker.C:
			resp, err := http.Get("http://" + addr + path)
			log.WithFields(log.Fields{
				"addr": addr,
				"path": path,
			}).Info("making_request")

			if err != nil {
				fmt.Println(err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode <= 200 {
				fmt.Printf("Service Ready code: %d", resp.StatusCode)
				return nil
			}

		case <-timer.C:
			return fmt.Errorf("timeout %s reached", timeout)
		}

	}
	return nil
}
