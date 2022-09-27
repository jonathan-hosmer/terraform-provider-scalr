package scalr

import (
	"context"
	"github.com/scalr/go-scalr"
	"testing"
)

func testResourceScalrWorkspaceStateDataV0() map[string]interface{} {
	return map[string]interface{}{
		"id":          "my-org/test",
		"external_id": "ws-123",
	}
}

func testResourceScalrWorkspaceStateDataV1() map[string]interface{} {
	v0 := testResourceScalrWorkspaceStateDataV0()
	return map[string]interface{}{
		"id": v0["external_id"],
	}
}

func TestResourceScalrWorkspaceStateUpgradeV0(t *testing.T) {
	expected := testResourceScalrWorkspaceStateDataV1()
	actual, err := resourceScalrWorkspaceStateUpgradeV0(testResourceScalrWorkspaceStateDataV0(), nil)
	assertCorrectState(t, err, actual, expected)
}

func testResourceScalrWorkspaceStateDataV1VcsRepo() map[string]interface{} {
	return map[string]interface{}{
		"id": "my-org/test",
		"vcs_repo": []interface{}{
			map[string]interface{}{
				"oauth_token_id": "test_provider_id",
				"identifier":     "test_identifier",
			},
		},
	}
}

func testResourceScalrWorkspaceStateDataV2() map[string]interface{} {
	v1 := testResourceScalrWorkspaceStateDataV1VcsRepo()
	vcsRepo := v1["vcs_repo"].([]interface{})[0].(map[string]interface{})
	return map[string]interface{}{
		"id": v1["id"],
		"vcs_repo": []interface{}{
			map[string]interface{}{
				"identifier": "test_identifier",
			},
		},
		"vcs_provider_id": vcsRepo["oauth_token_id"],
	}
}

func testResourceScalrWorkspaceStateDataV2NoVcs() map[string]interface{} {
	v1 := testResourceScalrWorkspaceStateDataV1()
	return map[string]interface{}{
		"id": v1["id"],
	}
}

func TestResourceScalrWorkspaceStateUpgradeV1(t *testing.T) {
	expected := testResourceScalrWorkspaceStateDataV2()
	actual, err := resourceScalrWorkspaceStateUpgradeV1(testResourceScalrWorkspaceStateDataV1VcsRepo(), nil)
	assertCorrectState(t, err, actual, expected)
}

func TestResourceScalrWorkspaceStateUpgradeV1NoVcs(t *testing.T) {
	expected := testResourceScalrWorkspaceStateDataV2NoVcs()
	actual, err := resourceScalrWorkspaceStateUpgradeV1(testResourceScalrWorkspaceStateDataV1(), nil)
	assertCorrectState(t, err, actual, expected)
}

func testResourceScalrWorkspaceStateDataV3() map[string]interface{} {
	v2 := testResourceScalrWorkspaceStateDataV2()
	delete(v2, "queue_all_runs")
	return v2
}

func TestResourceScalrWorkspaceStateUpgradeV2(t *testing.T) {
	expected := testResourceScalrWorkspaceStateDataV3()
	actual, err := resourceScalrWorkspaceStateUpgradeV2(testResourceScalrWorkspaceStateDataV2(), nil)
	assertCorrectState(t, err, actual, expected)
}

func testResourceScalrWorkspaceStateDataV3Operations() map[string]interface{} {
	return map[string]interface{}{
		"operations": false,
	}
}

func testResourceScalrWorkspaceStateDataV4Operations() map[string]interface{} {
	v3 := testResourceScalrWorkspaceStateDataV3Operations()

	var executionMode scalr.WorkspaceExecutionMode
	if v3["operations"].(bool) {
		executionMode = scalr.WorkspaceExecutionModeRemote
	} else {
		executionMode = scalr.WorkspaceExecutionModeLocal
	}

	return map[string]interface{}{
		"operations":     false,
		"execution_mode": executionMode,
	}
}

func TestResourceScalrWorkspaceStateUpgradeV3(t *testing.T) {
	expected := testResourceScalrWorkspaceStateDataV4Operations()
	actual, err := resourceScalrWorkspaceStateUpgradeV3(testResourceScalrWorkspaceStateDataV3Operations(), nil)
	assertCorrectState(t, err, actual, expected)
}

func testResourceScalrWorkspaceStateDataV4() map[string]interface{} {
	return map[string]interface{}{
		"id":             "ws-1234567890",
		"queue_all_runs": true,
	}
}

func testResourceScalrWorkspaceStateDataV4Before() map[string]interface{} {
	return map[string]interface{}{
		"id": "ws-1234567890",
	}
}

func TestResourceScalrWorkspaceStateUpgradeV4(t *testing.T) {
	client := testScalrClient(t)
	name := "test"
	client.Workspaces.Create(context.Background(), scalr.WorkspaceCreateOptions{
		ID:          "ws-1234567890",
		Name:        &name,
		Environment: &scalr.Environment{ID: "my-org"},
	})
	expected := testResourceScalrWorkspaceStateDataV4()
	actual, err := resourceScalrWorkspaceStateUpgradeV4(testResourceScalrWorkspaceStateDataV4Before(), client)
	assertCorrectState(t, err, actual, expected)
}
