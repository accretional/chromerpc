package integration

import (
	"context"
	"testing"
	"time"

	cachestoragepb "github.com/accretional/chromerpc/proto/cdp/cachestorage"
	consolepb "github.com/accretional/chromerpc/proto/cdp/console"
	heapprofilerpb "github.com/accretional/chromerpc/proto/cdp/heapprofiler"
	indexeddbpb "github.com/accretional/chromerpc/proto/cdp/indexeddb"
	pagepb "github.com/accretional/chromerpc/proto/cdp/page"
	serviceworkerpb "github.com/accretional/chromerpc/proto/cdp/serviceworker"
)

// =============================================================
// Console Domain Tests
// =============================================================

func TestConsoleEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.consoleClient.Enable(ctx, &consolepb.EnableRequest{})
	if err != nil {
		t.Fatalf("Console.Enable: %v", err)
	}

	_, err = env.consoleClient.Disable(ctx, &consolepb.DisableRequest{})
	if err != nil {
		t.Fatalf("Console.Disable: %v", err)
	}
}

func TestConsoleClearMessages(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.consoleClient.Enable(ctx, &consolepb.EnableRequest{})

	_, err := env.consoleClient.ClearMessages(ctx, &consolepb.ClearMessagesRequest{})
	if err != nil {
		t.Fatalf("Console.ClearMessages: %v", err)
	}
}

// =============================================================
// HeapProfiler Domain Tests
// =============================================================

func TestHeapProfilerEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.heapProfilerClient.Enable(ctx, &heapprofilerpb.EnableRequest{})
	if err != nil {
		t.Fatalf("HeapProfiler.Enable: %v", err)
	}

	_, err = env.heapProfilerClient.Disable(ctx, &heapprofilerpb.DisableRequest{})
	if err != nil {
		t.Fatalf("HeapProfiler.Disable: %v", err)
	}
}

func TestHeapProfilerCollectGarbage(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.heapProfilerClient.Enable(ctx, &heapprofilerpb.EnableRequest{})
	defer env.heapProfilerClient.Disable(ctx, &heapprofilerpb.DisableRequest{})

	_, err := env.heapProfilerClient.CollectGarbage(ctx, &heapprofilerpb.CollectGarbageRequest{})
	if err != nil {
		t.Fatalf("HeapProfiler.CollectGarbage: %v", err)
	}
}

func TestHeapProfilerSampling(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.heapProfilerClient.Enable(ctx, &heapprofilerpb.EnableRequest{})
	defer env.heapProfilerClient.Disable(ctx, &heapprofilerpb.DisableRequest{})

	_, err := env.heapProfilerClient.StartSampling(ctx, &heapprofilerpb.StartSamplingRequest{})
	if err != nil {
		t.Fatalf("HeapProfiler.StartSampling: %v", err)
	}

	// Generate some allocations.
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<script>var arr=[]; for(var i=0;i<100;i++) arr.push(new Array(1000));</script>",
	})
	time.Sleep(500 * time.Millisecond)

	stopResp, err := env.heapProfilerClient.StopSampling(ctx, &heapprofilerpb.StopSamplingRequest{})
	if err != nil {
		t.Fatalf("HeapProfiler.StopSampling: %v", err)
	}
	if stopResp.Profile == nil {
		t.Fatal("expected non-nil sampling profile")
	}
	if stopResp.Profile.Head == nil {
		t.Fatal("expected non-nil profile head")
	}
	t.Logf("Sampling profile: head selfSize=%.0f, samples=%d",
		stopResp.Profile.Head.SelfSize, len(stopResp.Profile.Samples))
}

func TestHeapProfilerGetHeapObjectId(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.heapProfilerClient.Enable(ctx, &heapprofilerpb.EnableRequest{})
	defer env.heapProfilerClient.Disable(ctx, &heapprofilerpb.DisableRequest{})

	// We need a valid objectId from Runtime to test this.
	// Just verify the RPC works by testing with an invalid ID.
	_, err := env.heapProfilerClient.GetHeapObjectId(ctx, &heapprofilerpb.GetHeapObjectIdRequest{
		ObjectId: "invalid-id",
	})
	if err != nil {
		t.Logf("GetHeapObjectId with invalid ID: %v (expected)", err)
	}
}

// =============================================================
// ServiceWorker Domain Tests
// =============================================================

func TestServiceWorkerEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.serviceWorkerClient.Enable(ctx, &serviceworkerpb.EnableRequest{})
	if err != nil {
		// ServiceWorker domain may not be available at browser level in all Chrome versions.
		t.Logf("ServiceWorker.Enable: %v (may require page-level session)", err)
		return
	}

	_, err = env.serviceWorkerClient.Disable(ctx, &serviceworkerpb.DisableRequest{})
	if err != nil {
		t.Fatalf("ServiceWorker.Disable: %v", err)
	}
}

func TestServiceWorkerSetForceUpdateOnPageLoad(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.serviceWorkerClient.Enable(ctx, &serviceworkerpb.EnableRequest{})
	if err != nil {
		t.Logf("ServiceWorker.Enable: %v (skipping)", err)
		return
	}
	defer env.serviceWorkerClient.Disable(ctx, &serviceworkerpb.DisableRequest{})

	_, err = env.serviceWorkerClient.SetForceUpdateOnPageLoad(ctx, &serviceworkerpb.SetForceUpdateOnPageLoadRequest{
		ForceUpdateOnPageLoad: true,
	})
	if err != nil {
		t.Logf("SetForceUpdateOnPageLoad: %v", err)
	}
}

func TestServiceWorkerStopAllWorkers(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.serviceWorkerClient.Enable(ctx, &serviceworkerpb.EnableRequest{})
	if err != nil {
		t.Logf("ServiceWorker.Enable: %v (skipping)", err)
		return
	}
	defer env.serviceWorkerClient.Disable(ctx, &serviceworkerpb.DisableRequest{})

	_, err = env.serviceWorkerClient.StopAllWorkers(ctx, &serviceworkerpb.StopAllWorkersRequest{})
	if err != nil {
		t.Logf("StopAllWorkers: %v", err)
	}
}

// =============================================================
// IndexedDB Domain Tests
// =============================================================

func TestIndexedDBEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.indexedDBClient.Enable(ctx, &indexeddbpb.EnableRequest{})
	if err != nil {
		t.Fatalf("IndexedDB.Enable: %v", err)
	}

	_, err = env.indexedDBClient.Disable(ctx, &indexeddbpb.DisableRequest{})
	if err != nil {
		t.Fatalf("IndexedDB.Disable: %v", err)
	}
}

func TestIndexedDBRequestDatabaseNames(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.indexedDBClient.Enable(ctx, &indexeddbpb.EnableRequest{})
	defer env.indexedDBClient.Disable(ctx, &indexeddbpb.DisableRequest{})

	// Navigate to a page to establish an origin.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>IndexedDB Test</h1>",
	})
	time.Sleep(300 * time.Millisecond)

	resp, err := env.indexedDBClient.RequestDatabaseNames(ctx, &indexeddbpb.RequestDatabaseNamesRequest{
		SecurityOrigin: "data://",
	})
	if err != nil {
		t.Logf("RequestDatabaseNames: %v (may fail for data:// origin)", err)
		return
	}
	t.Logf("Database names: %v", resp.DatabaseNames)
}

// =============================================================
// CacheStorage Domain Tests
// =============================================================

func TestCacheStorageRequestCacheNames(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Navigate to a page.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>CacheStorage Test</h1>",
	})
	time.Sleep(300 * time.Millisecond)

	origin := "data://"
	resp, err := env.cacheStorageClient.RequestCacheNames(ctx, &cachestoragepb.RequestCacheNamesRequest{
		SecurityOrigin: &origin,
	})
	if err != nil {
		t.Logf("RequestCacheNames: %v (may fail for data:// origin)", err)
		return
	}
	t.Logf("Caches: %d", len(resp.Caches))
	for _, c := range resp.Caches {
		t.Logf("  Cache: %s (id=%s)", c.CacheName, c.CacheId)
	}
}
