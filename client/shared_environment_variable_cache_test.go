package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func sharedEnvTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return New("test").WithBaseURL(server.URL).WithTeam(Team{ID: "team_default"})
}

func TestSharedEnvCacheConcurrentReads(t *testing.T) {
	var calls atomic.Int32
	c := sharedEnvTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, `{"id":"env_1","projectId":["project_1"],"target":["production"]}`)
	})
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range 30 {
		wg.Go(func() {
			<-start
			env, err := c.GetSharedEnvironmentVariable(context.Background(), "", "env_1")
			if err != nil {
				t.Error(err)
				return
			}
			if env.TeamID != "team_default" || env.ProjectIDs[0] != "project_1" {
				t.Errorf("unexpected response: %+v", env)
			}
			// Callers must not share mutable slices, including with the cache.
			env.ProjectIDs[0] = "changed"
			env.Target[0] = "changed"
		})
	}
	close(start)
	wg.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("GET calls = %d, want 1", got)
	}
	env, err := c.GetSharedEnvironmentVariable(context.Background(), "team_default", "env_1")
	if err != nil {
		t.Fatal(err)
	}
	if env.ProjectIDs[0] != "project_1" || env.Target[0] != "production" {
		t.Fatalf("cached response mutated: %+v", env)
	}
	if calls.Load() != 1 {
		t.Fatal("explicit default team should share the cache")
	}
}

func TestSharedEnvCacheIsolationAndExpiry(t *testing.T) {
	var calls atomic.Int32
	c := sharedEnvTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprintf(w, `{"id":%q}`, r.URL.Path)
	})
	ctx := context.Background()
	for _, key := range []sharedEnvKey{{"team_a", "env_1"}, {"team_b", "env_1"}, {"team_a", "env_2"}, {"team_a", "env_1"}} {
		if _, err := c.GetSharedEnvironmentVariable(ctx, key.teamID, key.envID); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
	entry, err := c.lockSharedEnv(ctx, "team_a", "env_1")
	if err != nil {
		t.Fatal(err)
	}
	entry.expires = time.Now().Add(-time.Second)
	entry.unlock()
	if _, err := c.GetSharedEnvironmentVariable(ctx, "team_a", "env_1"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 4 {
		t.Fatalf("expired entry wasn't fetched again: calls = %d", calls.Load())
	}
	other := New("other-token").WithBaseURL(c.baseURL)
	if _, err := other.GetSharedEnvironmentVariable(ctx, "team_a", "env_1"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 5 {
		t.Fatal("separate clients must not share cached responses")
	}
}

func TestSharedEnvCacheDoesNotCacheErrors(t *testing.T) {
	var calls atomic.Int32
	c := sharedEnvTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":{"code":"not_found","message":"missing"}}`)
			return
		}
		fmt.Fprint(w, `{"id":"env_1"}`)
	})
	if _, err := c.GetSharedEnvironmentVariable(context.Background(), "", "env_1"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.GetSharedEnvironmentVariable(context.Background(), "", "env_1"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

func TestSharedEnvCacheWriteInvalidation(t *testing.T) {
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		for _, failed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/failed=%t", method, failed), func(t *testing.T) {
				var reads atomic.Int32
				c := sharedEnvTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodGet {
						reads.Add(1)
						fmt.Fprint(w, `{"id":"env_1"}`)
						return
					}
					if r.Method != method {
						t.Errorf("method = %s, want %s", r.Method, method)
					}
					if failed {
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprint(w, `{"error":{"code":"internal_error","message":"failed"}}`)
					} else {
						fmt.Fprint(w, `{"updated":[{"id":"env_1"}]}`)
					}
				})
				ctx := context.Background()
				if _, err := c.GetSharedEnvironmentVariable(ctx, "", "env_1"); err != nil {
					t.Fatal(err)
				}
				var err error
				if method == http.MethodPatch {
					_, err = c.UpdateSharedEnvironmentVariable(ctx, UpdateSharedEnvironmentVariableRequest{TeamID: "team_default", EnvID: "env_1", ProjectIDUpdates: UpdateSharedEnvironmentVariableRequestProjectIDUpdates{Link: []string{"project_1"}}})
				} else {
					err = c.DeleteSharedEnvironmentVariable(ctx, "team_default", "env_1")
				}
				if (err != nil) != failed {
					t.Fatalf("write error = %v, failed = %t", err, failed)
				}
				if _, err := c.GetSharedEnvironmentVariable(ctx, "", "env_1"); err != nil {
					t.Fatal(err)
				}
				if reads.Load() != 2 {
					t.Fatalf("reads = %d, want 2", reads.Load())
				}
			})
		}
	}
}

func TestSharedEnvCacheReadOverlappingWrite(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var reads atomic.Int32
	var written atomic.Bool
	c := sharedEnvTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			written.Store(true)
			fmt.Fprint(w, `{"updated":[{"id":"env_1"}]}`)
			return
		}
		if reads.Add(1) == 1 {
			close(started)
			<-release
			fmt.Fprint(w, `{"id":"env_1","projectId":[]}`)
		} else {
			if !written.Load() {
				t.Error("second read preceded write")
			}
			fmt.Fprint(w, `{"id":"env_1","projectId":["project_1"]}`)
		}
	})
	ctx := context.Background()
	readDone := make(chan error, 1)
	go func() { _, err := c.GetSharedEnvironmentVariable(ctx, "", "env_1"); readDone <- err }()
	<-started
	writeDone := make(chan error, 1)
	go func() {
		_, err := c.UpdateSharedEnvironmentVariable(ctx, UpdateSharedEnvironmentVariableRequest{EnvID: "env_1", ProjectIDUpdates: UpdateSharedEnvironmentVariableRequestProjectIDUpdates{Link: []string{"project_1"}}})
		writeDone <- err
	}()
	close(release)
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	if err := <-writeDone; err != nil {
		t.Fatal(err)
	}
	env, err := c.GetSharedEnvironmentVariable(ctx, "", "env_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(env.ProjectIDs) != 1 {
		t.Fatal("read overlapping write left stale membership cached")
	}
	if reads.Load() != 2 {
		t.Fatalf("reads = %d, want 2", reads.Load())
	}
}

func TestSharedEnvCacheWaitingReadCancellation(t *testing.T) {
	c := New("test")
	entry, err := c.lockSharedEnv(context.Background(), "", "env_1")
	if err != nil {
		t.Fatal(err)
	}
	defer entry.unlock()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := c.GetSharedEnvironmentVariable(ctx, "", "env_1"); done <- err }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want cancellation", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waiting read ignored cancellation")
	}
}
