package tests

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInternalAuthorizationPolicyUsesGrpcPathsAndPreservesPublicReachability(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	chartDir := filepath.Dir(filepath.Dir(filename))
	var stderr bytes.Buffer
	dependency := exec.Command("helm", "dependency", "build", chartDir)
	dependency.Stderr = &stderr
	if err := dependency.Run(); err != nil {
		t.Fatalf("helm dependency build: %v: %s", err, stderr.String())
	}
	stderr.Reset()
	cmd := exec.Command("helm", "template", "egress", chartDir)
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("helm template: %v: %s", err, stderr.String())
	}
	rendered := string(output)
	if strings.Contains(rendered, "methods:") {
		t.Fatalf("authorization policy must not render gRPC full method names under methods:\n%s", rendered)
	}
	for _, path := range []string{
		"/agynio.api.egress.v1.EgressRulesService/ListEgressRulesByAgent",
		"/agynio.api.egress.v1.EgressRulesService/CountRulesReferencingSecret",
	} {
		if !strings.Contains(rendered, path) {
			t.Fatalf("rendered chart missing internal RPC path %s", path)
		}
	}
	if !strings.Contains(rendered, "paths:") {
		t.Fatal("authorization policy must render gRPC full method names under paths:")
	}
	if strings.Contains(rendered, "rules:\n    - {}") {
		t.Fatalf("authorization policy must not include an allow-all rule that bypasses internal RPC protection:\n%s", rendered)
	}
	if !strings.Contains(rendered, "notPaths:") {
		t.Fatalf("authorization policy must preserve public RPC reachability with notPaths:\n%s", rendered)
	}
}
