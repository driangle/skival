package verifier

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/driangle/skival/internal/suite"
)

// TCPProbeVerifier checks TCP connectivity to a host:port.
type TCPProbeVerifier struct {
	Probe suite.TCPProbe
}

func (v *TCPProbeVerifier) Verify(ctx context.Context, _ VerifyInput) VerifyResult {
	addr := fmt.Sprintf("%s:%d", v.Probe.Host, v.Probe.Port)

	timeout := 5 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}

	var d net.Dialer
	d.Timeout = timeout

	conn, err := d.DialContext(ctx, "tcp", addr)
	isOpen := err == nil
	if conn != nil {
		conn.Close()
	}

	a := v.Probe.Assert

	if a.Open != nil {
		if *a.Open && !isOpen {
			reason := fmt.Sprintf("expected tcp %s to be open, but connection failed", addr)
			if ctx.Err() != nil {
				reason = fmt.Sprintf("tcp probe timed out for %s: %v", addr, ctx.Err())
			}
			return VerifyResult{Pass: false, Reason: reason}
		}
		if !*a.Open && isOpen {
			return VerifyResult{
				Pass:   false,
				Reason: fmt.Sprintf("expected tcp %s to be closed, but connection succeeded", addr),
			}
		}
	}

	return VerifyResult{Pass: true, Reason: fmt.Sprintf("tcp probe passed for %s", addr)}
}
