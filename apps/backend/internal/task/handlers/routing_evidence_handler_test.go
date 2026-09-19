package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	dynamicruntime "github.com/kandev/kandev/internal/agent/runtime/dynamic"
)

type routingEvidenceFixture struct {
	result dynamicruntime.RoutingEvidenceResult
}

func (f routingEvidenceFixture) ListTaskRoutingEvidence(context.Context, string) (dynamicruntime.RoutingEvidenceResult, error) {
	return f.result, nil
}

func TestHTTPGetRoutingEvidenceIsAuthorizedAndReturnsPaidCount(t *testing.T) {
	h := newUsageTotalsHandlers(t, nil, nil)
	h.routingEvidence = routingEvidenceFixture{result: dynamicruntime.RoutingEvidenceResult{
		Attempts:            []dynamicruntime.RoutingEvidence{{AttemptID: "attempt-1", RouteClass: dynamicruntime.RouteClassStrong}},
		PaidEscalationCount: 1,
	}}

	params := gin.Params{{Key: "id", Value: "task-b"}}
	foreign, foreignRec := taskUsageRequestAs(t, "/api/v1/tasks/task-b/routing-evidence", "user-a", params)
	h.httpGetRoutingEvidence(foreign)
	if foreignRec.Code != http.StatusNotFound {
		t.Fatalf("foreign status = %d", foreignRec.Code)
	}

	owner, ownerRec := taskUsageRequestAs(t, "/api/v1/tasks/task-b/routing-evidence", "user-b", params)
	h.httpGetRoutingEvidence(owner)
	if ownerRec.Code != http.StatusOK {
		t.Fatalf("owner status = %d, body = %s", ownerRec.Code, ownerRec.Body.String())
	}
	var body dynamicruntime.RoutingEvidenceResult
	if err := json.Unmarshal(ownerRec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.PaidEscalationCount != 1 || len(body.Attempts) != 1 {
		t.Fatalf("body = %#v", body)
	}
}
