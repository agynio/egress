package server

import (
	"testing"

	egressv1 "github.com/agynio/egress/.gen/go/agynio/api/egress/v1"
	zitimanagementv1 "github.com/agynio/egress/.gen/go/agynio/api/ziti_management/v1"
	"github.com/agynio/egress/internal/store"
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
	if got := host.GetAllowedAddresses(); len(got) != 1 || got[0] != allIPv4Addresses {
		t.Fatalf("allowed addresses = %v", got)
	}
	if got := host.GetAllowedPortRanges(); len(got) != 1 || got[0].GetLow() != minimumTCPPort || got[0].GetHigh() != maximumTCPPort {
		t.Fatalf("allowed port ranges = %v", got)
	}
	intercept := req.GetInterceptV1Config()
	if got := intercept.GetAddresses(); len(got) != 1 || got[0] != "api.example.com" {
		t.Fatalf("intercept addresses = %v", got)
	}
}

func TestServiceMatchesRuleDetectsDrift(t *testing.T) {
	ruleID := uuid.New()
	rule := store.Rule{ID: ruleID, Matcher: &egressv1.EgressRuleMatcher{DomainPattern: "api.example.com", Ports: []int32{443}}}
	service := &zitimanagementv1.Service{
		ZitiServiceId:     "service-id",
		Name:              egressServiceName(ruleID),
		RoleAttributes:    []string{egressServiceRoleAttribute},
		HostV1Config:      hostV1Config(rule.Matcher),
		InterceptV1Config: interceptV1Config(rule.Matcher),
	}
	if !serviceMatchesRule(service, rule) {
		t.Fatal("expected service to match rule")
	}
	service.HostV1Config.AllowedAddresses = []string{"10.0.0.0/8"}
	if serviceMatchesRule(service, rule) {
		t.Fatal("expected host config drift to be detected")
	}
}

func TestServicePolicyMatchesAttachmentDetectsDrift(t *testing.T) {
	ruleID := uuid.New()
	agentID := uuid.New()
	attachment := store.Attachment{RuleID: ruleID, AgentID: agentID}
	policy := &zitimanagementv1.ServicePolicy{
		ZitiServicePolicyId: "policy-id",
		Name:                egressDialPolicyName(ruleID, agentID),
		Type:                zitimanagementv1.ServicePolicyType_SERVICE_POLICY_TYPE_DIAL,
		IdentityRoles:       []string{agentRole(agentID)},
		ServiceRoles:        []string{serviceNameRole(ruleID)},
	}
	if !servicePolicyMatchesAttachment(policy, attachment) {
		t.Fatal("expected policy to match attachment")
	}
	policy.IdentityRoles = []string{"#agent-drift"}
	if servicePolicyMatchesAttachment(policy, attachment) {
		t.Fatal("expected identity role drift to be detected")
	}
}
