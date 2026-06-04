package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	authorizationv1 "github.com/agynio/egress-rules/.gen/go/agynio/api/authorization/v1"
	egressv1 "github.com/agynio/egress-rules/.gen/go/agynio/api/egress/v1"
	notificationsv1 "github.com/agynio/egress-rules/.gen/go/agynio/api/notifications/v1"
	secretsv1 "github.com/agynio/egress-rules/.gen/go/agynio/api/secrets/v1"
	zitimanagementv1 "github.com/agynio/egress-rules/.gen/go/agynio/api/ziti_management/v1"
	"github.com/agynio/egress-rules/internal/store"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestCreateEgressRuleAttachmentRequiresAgentOrg(t *testing.T) {
	callerID := uuid.New()
	ruleID := uuid.New()
	agentID := uuid.New()
	organizationID := uuid.New()

	storeFake := &fakeRuleStore{rule: store.Rule{ID: ruleID, OrganizationID: organizationID, Matcher: &egressv1.EgressRuleMatcher{DomainPattern: "api.example.com"}, Effect: &egressv1.EgressRuleEffect{}}}
	authzFake := &fakeAuthorizationClient{allowed: map[string]bool{
		tupleKey(identityObject(callerID), organizationMemberRelation, organizationObject(organizationID)): true,
		tupleKey(identityObject(callerID), agentCanEditConfigRelation, agentObject(agentID)):               true,
		tupleKey(organizationObject(organizationID), agentOrgRelation, agentObject(agentID)):               false,
	}}
	zitiFake := &fakeZitiManagementClient{}

	srv := New(Options{Store: storeFake, AuthorizationClient: authzFake, NotificationsClient: fakeNotificationsClient{}, ZitiClient: zitiFake})
	_, err := srv.CreateEgressRuleAttachment(incomingIdentityContext(callerID), &egressv1.CreateEgressRuleAttachmentRequest{RuleId: ruleID.String(), AgentId: agentID.String()})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status = %v, err = %v", status.Code(err), err)
	}
	if zitiFake.createServicePolicyCalls != 0 {
		t.Fatalf("expected no Ziti provisioning before agent org check, got %d calls", zitiFake.createServicePolicyCalls)
	}
	if storeFake.createdAttachment != nil {
		t.Fatal("expected no attachment insert before agent org check")
	}
	if !authzFake.checked(tupleKey(organizationObject(organizationID), agentOrgRelation, agentObject(agentID))) {
		t.Fatal("expected org relation check for agent membership")
	}
}

func TestCreateEgressRuleAttachmentProvisionsAfterAgentOrgAllowed(t *testing.T) {
	callerID := uuid.New()
	ruleID := uuid.New()
	agentID := uuid.New()
	organizationID := uuid.New()

	storeFake := &fakeRuleStore{rule: store.Rule{ID: ruleID, OrganizationID: organizationID, Matcher: &egressv1.EgressRuleMatcher{DomainPattern: "api.example.com"}, Effect: &egressv1.EgressRuleEffect{}, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	authzFake := &fakeAuthorizationClient{allowed: map[string]bool{
		tupleKey(identityObject(callerID), organizationMemberRelation, organizationObject(organizationID)): true,
		tupleKey(identityObject(callerID), agentCanEditConfigRelation, agentObject(agentID)):               true,
		tupleKey(organizationObject(organizationID), agentOrgRelation, agentObject(agentID)):               true,
	}}
	zitiFake := &fakeZitiManagementClient{policyID: "policy-id"}

	srv := New(Options{Store: storeFake, AuthorizationClient: authzFake, NotificationsClient: fakeNotificationsClient{}, ZitiClient: zitiFake})
	resp, err := srv.CreateEgressRuleAttachment(incomingIdentityContext(callerID), &egressv1.CreateEgressRuleAttachmentRequest{RuleId: ruleID.String(), AgentId: agentID.String()})
	if err != nil {
		t.Fatalf("CreateEgressRuleAttachment: %v", err)
	}
	if resp.GetEgressRuleAttachment().GetRuleId() != ruleID.String() {
		t.Fatalf("rule id = %q", resp.GetEgressRuleAttachment().GetRuleId())
	}
	if zitiFake.createServicePolicyCalls != 1 {
		t.Fatalf("ziti policy calls = %d", zitiFake.createServicePolicyCalls)
	}
	if got := zitiFake.lastPolicy.GetServiceRoles(); len(got) != 1 || got[0] != serviceNameRole(ruleID) {
		t.Fatalf("service roles = %v", got)
	}
}

func incomingIdentityContext(identityID uuid.UUID) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(identityMetadata, identityID.String()))
}

func tupleKey(user string, relation string, object string) string {
	return fmt.Sprintf("%s|%s|%s", user, relation, object)
}

type fakeAuthorizationClient struct {
	allowed map[string]bool
	checks  []string
}

func (f *fakeAuthorizationClient) Check(_ context.Context, req *authorizationv1.CheckRequest, _ ...grpc.CallOption) (*authorizationv1.CheckResponse, error) {
	key := req.GetTupleKey()
	value := tupleKey(key.GetUser(), key.GetRelation(), key.GetObject())
	f.checks = append(f.checks, value)
	return &authorizationv1.CheckResponse{Allowed: f.allowed[value]}, nil
}

func (f *fakeAuthorizationClient) checked(expected string) bool {
	for _, check := range f.checks {
		if check == expected {
			return true
		}
	}
	return false
}

type fakeRuleStore struct {
	rule              store.Rule
	rules             []store.Rule
	attachments       []store.Attachment
	updatedServiceID  string
	updatedPolicyID   string
	createdAttachment *store.Attachment
}

func (f *fakeRuleStore) CreateRule(context.Context, store.Rule) error { return nil }
func (f *fakeRuleStore) UpdateRule(context.Context, store.Rule) error { return nil }
func (f *fakeRuleStore) UpdateRuleServiceID(_ context.Context, _ uuid.UUID, serviceID string) error {
	f.updatedServiceID = serviceID
	return nil
}
func (f *fakeRuleStore) GetRule(context.Context, uuid.UUID) (store.Rule, error) { return f.rule, nil }
func (f *fakeRuleStore) ListRules(context.Context, uuid.UUID, int32, *store.PageCursor) (store.RuleListResult, error) {
	return store.RuleListResult{}, nil
}
func (f *fakeRuleStore) ListAllRules(context.Context) ([]store.Rule, error) {
	return f.rules, nil
}
func (f *fakeRuleStore) ListRulesByAgent(context.Context, uuid.UUID) ([]store.Rule, error) {
	return nil, nil
}
func (f *fakeRuleStore) DeleteRule(context.Context, uuid.UUID) error { return nil }
func (f *fakeRuleStore) CountAttachmentsByRule(context.Context, uuid.UUID) (int32, error) {
	return 0, nil
}
func (f *fakeRuleStore) CreateAttachment(_ context.Context, attachment store.Attachment) error {
	f.createdAttachment = &attachment
	return nil
}
func (f *fakeRuleStore) UpdateAttachmentPolicyID(_ context.Context, _ uuid.UUID, policyID string) error {
	f.updatedPolicyID = policyID
	return nil
}
func (f *fakeRuleStore) GetAttachment(context.Context, uuid.UUID) (store.Attachment, error) {
	if f.createdAttachment == nil {
		return store.Attachment{}, store.ErrAttachmentNotFound
	}
	attachment := *f.createdAttachment
	attachment.CreatedAt = time.Now()
	attachment.UpdatedAt = attachment.CreatedAt
	return attachment, nil
}
func (f *fakeRuleStore) ListAllAttachments(context.Context) ([]store.Attachment, error) {
	return f.attachments, nil
}
func (f *fakeRuleStore) ListAttachments(context.Context, uuid.UUID, *uuid.UUID, *uuid.UUID, int32, *store.PageCursor) (store.AttachmentListResult, error) {
	return store.AttachmentListResult{}, nil
}
func (f *fakeRuleStore) DeleteAttachment(context.Context, uuid.UUID) error { return nil }
func (f *fakeRuleStore) CountRulesReferencingSecret(context.Context, uuid.UUID) (int32, []uuid.UUID, error) {
	return 0, nil, nil
}

type fakeZitiManagementClient struct {
	policyID                 string
	createServicePolicyCalls int
	lastPolicy               *zitimanagementv1.CreateServicePolicyRequest
}

func (f *fakeZitiManagementClient) CreateService(context.Context, *zitimanagementv1.CreateServiceRequest, ...grpc.CallOption) (*zitimanagementv1.CreateServiceResponse, error) {
	return &zitimanagementv1.CreateServiceResponse{ZitiServiceId: "service-id"}, nil
}
func (f *fakeZitiManagementClient) GetService(context.Context, *zitimanagementv1.GetServiceRequest, ...grpc.CallOption) (*zitimanagementv1.GetServiceResponse, error) {
	return &zitimanagementv1.GetServiceResponse{Service: &zitimanagementv1.Service{ZitiServiceId: "service-id"}}, nil
}
func (f *fakeZitiManagementClient) ListServices(context.Context, *zitimanagementv1.ListServicesRequest, ...grpc.CallOption) (*zitimanagementv1.ListServicesResponse, error) {
	return &zitimanagementv1.ListServicesResponse{}, nil
}
func (f *fakeZitiManagementClient) UpdateService(context.Context, *zitimanagementv1.UpdateServiceRequest, ...grpc.CallOption) (*zitimanagementv1.UpdateServiceResponse, error) {
	return &zitimanagementv1.UpdateServiceResponse{Service: &zitimanagementv1.Service{ZitiServiceId: "service-id"}}, nil
}
func (f *fakeZitiManagementClient) DeleteService(context.Context, *zitimanagementv1.DeleteServiceRequest, ...grpc.CallOption) (*zitimanagementv1.DeleteServiceResponse, error) {
	return &zitimanagementv1.DeleteServiceResponse{}, nil
}
func (f *fakeZitiManagementClient) CreateServicePolicy(_ context.Context, req *zitimanagementv1.CreateServicePolicyRequest, _ ...grpc.CallOption) (*zitimanagementv1.CreateServicePolicyResponse, error) {
	f.createServicePolicyCalls++
	f.lastPolicy = req
	return &zitimanagementv1.CreateServicePolicyResponse{ZitiServicePolicyId: f.policyID}, nil
}
func (f *fakeZitiManagementClient) GetServicePolicy(context.Context, *zitimanagementv1.GetServicePolicyRequest, ...grpc.CallOption) (*zitimanagementv1.GetServicePolicyResponse, error) {
	return &zitimanagementv1.GetServicePolicyResponse{ServicePolicy: &zitimanagementv1.ServicePolicy{ZitiServicePolicyId: f.policyID}}, nil
}
func (f *fakeZitiManagementClient) ListServicePolicies(context.Context, *zitimanagementv1.ListServicePoliciesRequest, ...grpc.CallOption) (*zitimanagementv1.ListServicePoliciesResponse, error) {
	return &zitimanagementv1.ListServicePoliciesResponse{}, nil
}
func (f *fakeZitiManagementClient) DeleteServicePolicy(context.Context, *zitimanagementv1.DeleteServicePolicyRequest, ...grpc.CallOption) (*zitimanagementv1.DeleteServicePolicyResponse, error) {
	return &zitimanagementv1.DeleteServicePolicyResponse{}, nil
}

type fakeNotificationsClient struct{}

func (fakeNotificationsClient) Publish(context.Context, *notificationsv1.PublishRequest, ...grpc.CallOption) (*notificationsv1.PublishResponse, error) {
	return &notificationsv1.PublishResponse{}, nil
}

type fakeSecretsClient struct{}

func (fakeSecretsClient) ResolveSecretExists(context.Context, *secretsv1.ResolveSecretExistsRequest, ...grpc.CallOption) (*secretsv1.ResolveSecretExistsResponse, error) {
	return &secretsv1.ResolveSecretExistsResponse{Exists: true}, nil
}
