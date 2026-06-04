package server

import (
	"context"
	"fmt"

	egressv1 "github.com/agynio/egress/.gen/go/agynio/api/egress/v1"
	zitimanagementv1 "github.com/agynio/egress/.gen/go/agynio/api/ziti_management/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	egressServiceRoleAttribute = "egress-services"
	tcpProtocol                = "tcp"
)

func egressServiceName(ruleID uuid.UUID) string {
	return fmt.Sprintf("egress-rule-%s", ruleID)
}

func egressDialPolicyName(ruleID uuid.UUID, agentID uuid.UUID) string {
	return fmt.Sprintf("egress-rule-%s-agent-%s-dial", ruleID, agentID)
}

func agentRole(agentID uuid.UUID) string {
	return fmt.Sprintf("#agent-%s", agentID)
}

func serviceNameRole(ruleID uuid.UUID) string {
	return fmt.Sprintf("@%s", egressServiceName(ruleID))
}

func (s *Server) provisionRuleService(ctx context.Context, ruleID uuid.UUID, matcher *egressv1.EgressRuleMatcher) (string, error) {
	resp, err := s.zitiClient.CreateService(ctx, createServiceRequest(ruleID, matcher))
	if err != nil {
		return "", status.Errorf(codes.Internal, "create egress rule service: %v", err)
	}
	serviceID := resp.GetZitiServiceId()
	if serviceID == "" {
		return "", status.Error(codes.Internal, "create egress rule service: missing ziti_service_id")
	}
	return serviceID, nil
}

func (s *Server) deleteRuleService(ctx context.Context, serviceID string) error {
	if serviceID == "" {
		return nil
	}
	_, err := s.zitiClient.DeleteService(ctx, &zitimanagementv1.DeleteServiceRequest{ZitiServiceId: serviceID})
	if err != nil {
		return status.Errorf(codes.Internal, "delete egress rule service: %v", err)
	}
	return nil
}

func (s *Server) provisionAttachmentPolicy(ctx context.Context, ruleID uuid.UUID, agentID uuid.UUID) (string, error) {
	resp, err := s.zitiClient.CreateServicePolicy(ctx, &zitimanagementv1.CreateServicePolicyRequest{
		Type:          zitimanagementv1.ServicePolicyType_SERVICE_POLICY_TYPE_DIAL,
		Name:          egressDialPolicyName(ruleID, agentID),
		IdentityRoles: []string{agentRole(agentID)},
		ServiceRoles:  []string{serviceNameRole(ruleID)},
	})
	if err != nil {
		return "", status.Errorf(codes.Internal, "create egress rule dial policy: %v", err)
	}
	policyID := resp.GetZitiServicePolicyId()
	if policyID == "" {
		return "", status.Error(codes.Internal, "create egress rule dial policy: missing ziti_service_policy_id")
	}
	return policyID, nil
}

func (s *Server) deleteAttachmentPolicy(ctx context.Context, policyID string) error {
	if policyID == "" {
		return nil
	}
	_, err := s.zitiClient.DeleteServicePolicy(ctx, &zitimanagementv1.DeleteServicePolicyRequest{ZitiServicePolicyId: policyID})
	if err != nil {
		return status.Errorf(codes.Internal, "delete egress rule dial policy: %v", err)
	}
	return nil
}

func createServiceRequest(ruleID uuid.UUID, matcher *egressv1.EgressRuleMatcher) *zitimanagementv1.CreateServiceRequest {
	portRanges := portRangesFromPorts(matcher.GetPorts())
	return &zitimanagementv1.CreateServiceRequest{
		Name:           egressServiceName(ruleID),
		RoleAttributes: []string{egressServiceRoleAttribute},
		HostV1Config: &zitimanagementv1.HostV1Config{
			Protocol:          tcpProtocol,
			ForwardProtocol:   true,
			ForwardAddress:    true,
			ForwardPort:       true,
			AllowedProtocols:  []string{tcpProtocol},
			AllowedAddresses:  []string{matcher.GetDomainPattern()},
			AllowedPortRanges: portRanges,
		},
		InterceptV1Config: &zitimanagementv1.InterceptV1Config{
			Protocols:  []string{tcpProtocol},
			Addresses:  []string{matcher.GetDomainPattern()},
			PortRanges: portRanges,
		},
	}
}

func portRangesFromPorts(ports []int32) []*zitimanagementv1.PortRange {
	ranges := make([]*zitimanagementv1.PortRange, 0, len(ports))
	for _, port := range ports {
		ranges = append(ranges, &zitimanagementv1.PortRange{Low: port, High: port})
	}
	return ranges
}
