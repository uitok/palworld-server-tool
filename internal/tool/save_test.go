package tool

import (
	"errors"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"testing"

	"github.com/zaigie/palworld-server-tool/internal/source"
)

func resetSourceHooks() {
	downloadFromHTTP = source.DownloadFromHttp
	parseK8sAddress = source.ParseK8sAddress
	copyFromPod = source.CopyFromPod
	parseDockerAddress = source.ParseDockerAddress
	copyFromContainer = source.CopyFromContainer
	copyFromLocal = source.CopyFromLocal
}

func TestGetFromSourceSelectsHandlersByPrefix(t *testing.T) {
	defer resetSourceHooks()

	t.Run("http", func(t *testing.T) {
		called := false
		downloadFromHTTP = func(url, way string) (string, error) {
			called = true
			if url != "https://example.com/sav.zip" || way != "backup" {
				t.Fatalf("unexpected http args: %s %s", url, way)
			}
			return "/tmp/http/Level.sav", nil
		}

		got, err := getFromSource("https://example.com/sav.zip", "backup")
		if err != nil {
			t.Fatalf("expected http source to succeed, got %v", err)
		}
		if !called || got != "/tmp/http/Level.sav" {
			t.Fatalf("unexpected http result: called=%v path=%s", called, got)
		}
	})

	resetSourceHooks()
	t.Run("k8s", func(t *testing.T) {
		parsed := false
		copied := false
		parseK8sAddress = func(address string) (string, string, string, string, error) {
			parsed = true
			if address != "k8s://ns/pod/container:/data" {
				t.Fatalf("unexpected k8s address: %s", address)
			}
			return "ns", "pod", "container", "/data", nil
		}
		copyFromPod = func(namespace, podName, container, remotePath, way string) (string, error) {
			copied = true
			if namespace != "ns" || podName != "pod" || container != "container" || remotePath != "/data" || way != "decode" {
				t.Fatalf("unexpected k8s copy args: %s %s %s %s %s", namespace, podName, container, remotePath, way)
			}
			return "/tmp/k8s/Level.sav", nil
		}

		got, err := getFromSource("k8s://ns/pod/container:/data", "decode")
		if err != nil {
			t.Fatalf("expected k8s source to succeed, got %v", err)
		}
		if !parsed || !copied || got != "/tmp/k8s/Level.sav" {
			t.Fatalf("unexpected k8s result: parsed=%v copied=%v path=%s", parsed, copied, got)
		}
	})

	resetSourceHooks()
	t.Run("docker", func(t *testing.T) {
		parsed := false
		copied := false
		parseDockerAddress = func(address string) (string, string, error) {
			parsed = true
			if address != "docker://container:/data" {
				t.Fatalf("unexpected docker address: %s", address)
			}
			return "container", "/data", nil
		}
		copyFromContainer = func(containerID, remotePath, way string) (string, error) {
			copied = true
			if containerID != "container" || remotePath != "/data" || way != "backup" {
				t.Fatalf("unexpected docker copy args: %s %s %s", containerID, remotePath, way)
			}
			return "/tmp/docker/Level.sav", nil
		}

		got, err := getFromSource("docker://container:/data", "backup")
		if err != nil {
			t.Fatalf("expected docker source to succeed, got %v", err)
		}
		if !parsed || !copied || got != "/tmp/docker/Level.sav" {
			t.Fatalf("unexpected docker result: parsed=%v copied=%v path=%s", parsed, copied, got)
		}
	})

	resetSourceHooks()
	t.Run("local", func(t *testing.T) {
		called := false
		copyFromLocal = func(src, way string) (string, error) {
			called = true
			if src != "/srv/palworld/Level.sav" || way != "decode" {
				t.Fatalf("unexpected local args: %s %s", src, way)
			}
			return "/tmp/local/Level.sav", nil
		}

		got, err := getFromSource("/srv/palworld/Level.sav", "decode")
		if err != nil {
			t.Fatalf("expected local source to succeed, got %v", err)
		}
		if !called || got != "/tmp/local/Level.sav" {
			t.Fatalf("unexpected local result: called=%v path=%s", called, got)
		}
	})
}

func TestGetFromSourceWrapsErrors(t *testing.T) {
	defer resetSourceHooks()

	downloadFromHTTP = func(url, way string) (string, error) {
		return "", &neturl.Error{Op: http.MethodGet, URL: url, Err: stubNetError{}}
	}
	if _, err := getFromSource("http://example.com/sav.zip", "backup"); err == nil || SaveOperationErrorCode(err) != "save_source_unreachable" {
		t.Fatalf("expected unreachable http error, got %v (%s)", err, SaveOperationErrorCode(err))
	}

	parseK8sAddress = func(address string) (string, string, string, string, error) {
		return "", "", "", "", errors.New("bad k8s")
	}
	if _, err := getFromSource("k8s://bad", "backup"); err == nil || SaveOperationErrorCode(err) != "save_source_invalid" {
		t.Fatalf("expected invalid k8s parse error, got %v (%s)", err, SaveOperationErrorCode(err))
	}

	resetSourceHooks()
	parseDockerAddress = func(address string) (string, string, error) { return "", "", errors.New("bad docker") }
	if _, err := getFromSource("docker://bad", "backup"); err == nil || SaveOperationErrorCode(err) != "save_source_invalid" {
		t.Fatalf("expected invalid docker parse error, got %v (%s)", err, SaveOperationErrorCode(err))
	}

	resetSourceHooks()
	copyFromLocal = func(src, way string) (string, error) { return "", os.ErrNotExist }
	if _, err := getFromSource("/bad/local", "backup"); err == nil || SaveOperationErrorCode(err) != "save_source_not_found" {
		t.Fatalf("expected not found local error, got %v (%s)", err, SaveOperationErrorCode(err))
	}
}

func TestSaveOperationErrorHelpers(t *testing.T) {
	err := wrapSaveDecodeError("/srv/palworld/Level.sav", "cli", os.ErrNotExist)
	if SaveOperationErrorCode(err) != "save_decode_cli_missing" {
		t.Fatalf("expected save_decode_cli_missing, got %s", SaveOperationErrorCode(err))
	}
	details := SaveOperationErrorDetails(err)
	if details == nil || details["stage"] != "cli" || details["source_kind"] != SaveSourceLocal {
		t.Fatalf("unexpected error details: %#v", details)
	}

	err = wrapSaveSourceError("https://example.com/sav.zip", "download", errors.New("Level.sav not found"))
	if SaveOperationErrorCode(err) != "save_source_invalid" {
		t.Fatalf("expected invalid http content error, got %s", SaveOperationErrorCode(err))
	}
	if !strings.Contains(err.Error(), "invalid http save source content") {
		t.Fatalf("expected descriptive http content error, got %v", err)
	}
}
