package server

import (
	"testing"

	egressv1 "github.com/agynio/egress-rules/.gen/go/agynio/api/egress/v1"
	"github.com/google/uuid"
)

func TestCreateServiceRequestUsesForwardingHostConfig(t *testing.T) {
	ruleID := uuid.New()
	req := createServiceRequest(ruleID, &egressv1.EgressRuleMatcher{DomainPattern: "api.example.com", Ports: []int32{443}})
	if req.GetName() != "egress-rule-"+ruleID.String() {
		t.Fatalf("name = %q", req.GetName())
	}
	if len(req.GetRoleAttributes()) != 1 || req.GetRoleAttributes()[0] != "egress-services" {
		t.Fatalf("role attrs = %v", req.GetRoleAttributes())
	}
	host := req.GetHostV1Config()
	if host == nil {
		t.Fatal("missing host config")
	}
	if !host.GetForwardAddress() || !host.GetForwardPort() || !host.GetForwardProtocol() {
		t.Fatalf("forwarding flags not all enabled: %+v", host)
	}
	if got := host.GetAllowedAddresses(); len(got) != 1 || got[0] != "api.example.com" {
		t.Fatalf("allowed addresses = %v", got)
	}
	if got := host.GetAllowedPortRanges(); len(got) != 1 || got[0].GetLow() != 443 || got[0].GetHigh() != 443 {
		t.Fatalf("allowed port ranges = %v", got)
	}
	intercept := req.GetInterceptV1Config()
	if got := intercept.GetAddresses(); len(got) != 1 || got[0] != "api.example.com" {
		t.Fatalf("intercept addresses = %v", got)
	}
}
