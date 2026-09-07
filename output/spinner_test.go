package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSpinner_NonInteractivePrintsMessageOnce(t *testing.T) {
	var buf bytes.Buffer
	s := newSpinner("Setting up console", &buf, false)
	s.Start()
	s.Succeed("Console ready")

	out := buf.String()
	if !strings.Contains(out, "Setting up console") {
		t.Fatalf("expected initial message, got %q", out)
	}
	if !strings.Contains(out, "✓ Console ready") {
		t.Fatalf("expected success line, got %q", out)
	}
	if strings.Contains(out, "\r") {
		t.Fatalf("non-interactive output must not use carriage returns, got %q", out)
	}
}

func TestSpinner_Fail(t *testing.T) {
	var buf bytes.Buffer
	s := newSpinner("Setting up console", &buf, false)
	s.Start()
	s.Fail("Could not start the console pod")

	if !strings.Contains(buf.String(), "✗ Could not start the console pod") {
		t.Fatalf("expected failure line, got %q", buf.String())
	}
}

func TestSpinner_InteractiveAnimatesAndClears(t *testing.T) {
	var buf bytes.Buffer
	s := newSpinner("Working", &buf, true)
	s.Start()
	time.Sleep(250 * time.Millisecond)
	s.Succeed("Done")

	out := buf.String()
	if !strings.Contains(out, "\r") {
		t.Fatalf("interactive output should redraw with carriage returns, got %q", out)
	}
	if !strings.Contains(out, "Working") {
		t.Fatalf("expected spinner message, got %q", out)
	}
	if !strings.HasSuffix(out, "✓ Done\n") {
		t.Fatalf("expected final success line ending, got %q", out)
	}
}

func TestSpinner_StopIsIdempotent(t *testing.T) {
	var buf bytes.Buffer
	s := newSpinner("Working", &buf, false)
	s.Start()
	s.Succeed("Done")
	s.Fail("should not appear")

	if strings.Contains(buf.String(), "should not appear") {
		t.Fatalf("stop must be idempotent, got %q", buf.String())
	}
}

func TestSpinner_UpdateMessage(t *testing.T) {
	var buf bytes.Buffer
	s := newSpinner("Setting up", &buf, false)
	s.Start()
	s.UpdateMessage("Connecting to app")
	s.Stop()

	out := buf.String()
	if !strings.Contains(out, "Connecting to app") {
		t.Fatalf("expected updated message, got %q", out)
	}
	if strings.Contains(out, "✓") || strings.Contains(out, "✗") {
		t.Fatalf("Stop must not print a mark, got %q", out)
	}
}

func TestSpinner_StopClearsInteractiveLine(t *testing.T) {
	var buf bytes.Buffer
	s := newSpinner("Working", &buf, true)
	s.Start()
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	out := buf.String()
	if !strings.HasSuffix(out, "\r\033[K") {
		t.Fatalf("expected trailing clear sequence, got %q", out)
	}
}

func TestPrinterSpinner_WritesToTheErrorStream(t *testing.T) {
	pr, buf := newTestPrinter()

	s := pr.Spinner("Working")
	s.Start()
	s.Succeed("Done")

	if !strings.Contains(buf.String(), "✓ Done") {
		t.Fatalf("printer spinner did not write to Err, got %q", buf.String())
	}
}
