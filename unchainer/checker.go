package unchainer

import (
	"context"
	"fmt"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/network"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/rpcc"
	"golang.org/x/sync/errgroup"
	"time"
)

// Checker interface for link checkers
type Checker interface {
	check(link string) Result
}

// LinkChecker provides everything to check links
type LinkChecker struct {
	ctx               context.Context
	cancel            context.CancelFunc
	devtool           devtool.DevTools
	conn              rpcc.Conn
	client            cdp.Client
	responseReceived  network.ResponseReceivedClient
	requestWillBeSent network.RequestWillBeSentClient
	handler           func(recv chan<- string, done chan<- bool)
	destroyChecker    chan bool
}

// InitChecker create connections, error handlers and event handlers
func InitChecker(devtoolURL string, redirectWaitTime time.Duration) (*LinkChecker, error) {
	lc := LinkChecker{}

	lc.ctx, lc.cancel = context.WithCancel(context.Background())

	devt := devtool.New(devtoolURL)
	pt, err := devt.Get(lc.ctx, devtool.Page)
	if err != nil {
		pt, err = devt.Create(lc.ctx)
		if err != nil {
			return nil, err
		}
	}

	conn, err := rpcc.DialContext(lc.ctx, pt.WebSocketDebuggerURL)
	if err != nil {
		return nil, err
	}
	lc.conn = *conn

	lc.client = *cdp.NewClient(conn)

	// Give enough capacity to avoid blocking any event listeners
	abort := make(chan error, 5)
	lc.destroyChecker = make(chan bool)

	// Watch the abort channel.
	go func() {
		select {
		case <-lc.ctx.Done():
			return
		case err := <-abort:
			if err != nil {
				fmt.Printf("aborted: %s\n", err.Error())
			}
			lc.cancel()
			return
		case <-lc.destroyChecker:
			return
		}
	}()

	// Setup event handlers early because domain events can be sent as
	// soon as Enable is called on the domain.
	if err = lc.abortOnErrors(abort); err != nil {
		return nil, err
	}

	//RequestWillBeSent handles 301 redirects
	requestWillBeSent, err := lc.client.Network.RequestWillBeSent(lc.ctx)
	if err != nil {
		abort <- err
	}
	lc.requestWillBeSent = requestWillBeSent

	//ResponseReceived is used to trace HTML and JS redirects
	responseReceived, err := lc.client.Network.ResponseReceived(lc.ctx)
	if err != nil {
		abort <- err
	}
	lc.responseReceived = responseReceived

	if err = runBatch(
		// Enable all the domain events that we're interested in.
		func() error { return lc.client.Network.Enable(lc.ctx, nil) },
		func() error { return lc.client.Runtime.Enable(lc.ctx) },
	); err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Get ready to receive events
	lc.handler = func(recv chan<- string, done chan<- bool) {
		for {
			select {
			case <-requestWillBeSent.Ready():
				ev, err := requestWillBeSent.Recv()
				if err != nil {
					abort <- err
					return
				}

				if ev.RedirectResponse != nil {
					recv <- ev.RedirectResponse.URL
				}

			case <-responseReceived.Ready():
				ev, err := responseReceived.Recv()
				if err != nil {
					abort <- err
					return
				}

				recv <- ev.Response.URL

			case <-time.After(redirectWaitTime):
				// enough waiting, let other do their job
				done <- true
				return
			case <-lc.destroyChecker:
				// stop everything
				done <- true
				return
			}

		}
	}

	return &lc, nil
}

// Check main method
func (lc *LinkChecker) Check(link string) (*Result, error) {
	recv := make(chan string, 2)
	done := make(chan bool, 2)
	// Start handler to be ready to accept events notifying redirects
	go lc.handler(recv, done)
	// Navigate to starting page to trigger other events
	_, err := lc.client.Page.Navigate(lc.ctx, page.NewNavigateArgs(link))
	if err != nil {
		return nil, err
	}
	var result Result

	result.Start = link

loop:
	for {
		select {
		case <-done:
			close(recv)
			close(done)
			break loop
		case res := <-recv:
			// gather all the responses
			result.Chain = append(result.Chain, res)
		}
	}
	return &result, err
}

// Close clear all the mess after work
func (lc *LinkChecker) Close() {
	lc.destroyChecker <- true
	lc.requestWillBeSent.Close()
	lc.responseReceived.Close()
	lc.conn.Close()
	lc.cancel()
	close(lc.destroyChecker)
}

// abourOnErrors be ready to handle critical errors and stop working properly
func (lc *LinkChecker) abortOnErrors(abort chan<- error) error {
	exceptionThrown, err := lc.client.Runtime.ExceptionThrown(lc.ctx)
	if err != nil {
		return err
	}

	loadingFailed, err := lc.client.Network.LoadingFailed(lc.ctx)
	if err != nil {
		return err
	}

	go func() {
		defer exceptionThrown.Close()
		defer loadingFailed.Close()
		for {
			select {
			case <-exceptionThrown.Ready():
				ev, err := exceptionThrown.Recv()
				if err != nil {
					abort <- err
					return
				}

				abort <- ev.ExceptionDetails

			case <-loadingFailed.Ready():
				ev, err := loadingFailed.Recv()
				if err != nil {
					abort <- err
					return
				}
				canceled := ev.Canceled != nil && *ev.Canceled

				if !canceled {
					abort <- fmt.Errorf("request %s failed: %s", ev.RequestID, ev.ErrorText)
				}

			case <-lc.destroyChecker:
				close(abort)
				return
			}

		}
	}()
	return nil
}

// runBatchFunc type for signature for functions which could be run in batches
type runBatchFunc func() error

// runBatch run all the funcs and gather errors they produce
func runBatch(fn ...runBatchFunc) error {
	eg := errgroup.Group{}
	for _, f := range fn {
		eg.Go(f)
	}
	return eg.Wait()
}
