package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/maraino/retry"
)

func main() {

	executor := retry.NewExecutor().
		WithRetries(2).
		WithBackoff(retry.ExponentialDelayBackoff(500*time.Millisecond, 2))

	var robots []byte
	err := executor.Execute(func() error {
		res, err := http.Get("http://www.google.com/robots.txt")
		if err != nil {
			return err
		}

		robots, err = ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return err
		}
		return nil
	})

	if err == nil {
		fmt.Printf("%s", robots)
	}
}
