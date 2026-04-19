package verifier

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/driangle/skival/internal/suite"
)

func TestTCPProbeVerifier_OpenPortPasses(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	v := &TCPProbeVerifier{
		Probe: suite.TCPProbe{
			Host:   "127.0.0.1",
			Port:   port,
			Assert: suite.TCPProbeAssert{Open: boolPtrVal(true)},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass: %s", result.Reason)
	}
}

func TestTCPProbeVerifier_ClosedPortFails(t *testing.T) {
	v := &TCPProbeVerifier{
		Probe: suite.TCPProbe{
			Host:   "127.0.0.1",
			Port:   1, // unlikely to be open
			Assert: suite.TCPProbeAssert{Open: boolPtrVal(true)},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on closed port")
	}
}

func TestTCPProbeVerifier_ClosedExpected(t *testing.T) {
	v := &TCPProbeVerifier{
		Probe: suite.TCPProbe{
			Host:   "127.0.0.1",
			Port:   1,
			Assert: suite.TCPProbeAssert{Open: boolPtrVal(false)},
		},
	}
	result := v.Verify(context.Background(), VerifyInput{})
	if !result.Pass {
		t.Fatalf("expected pass when port is closed and expected closed: %s", result.Reason)
	}
}

func TestTCPProbeVerifier_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Use a non-routable address to trigger timeout
	v := &TCPProbeVerifier{
		Probe: suite.TCPProbe{
			Host:   "192.0.2.1", // TEST-NET, non-routable
			Port:   9999,
			Assert: suite.TCPProbeAssert{Open: boolPtrVal(true)},
		},
	}
	result := v.Verify(ctx, VerifyInput{})
	if result.Pass {
		t.Fatal("expected fail on timeout")
	}
}

func TestTCPProbeVerifier_ImplementsVerifier(t *testing.T) {
	var _ Verifier = &TCPProbeVerifier{}
}
