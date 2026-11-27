package monitor

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
)

func TestAccountsListAndImportsExports(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	m := monitorFor(t, c, "sys")
	connect(t, c, 1, "admin")

	list, err := m.Accounts()
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]AccountRow{}
	for _, a := range list.Accounts {
		byName[a.Name] = a
	}
	for _, want := range []string{"APP", "USER", "SYS"} {
		if _, ok := byName[want]; !ok {
			t.Fatalf("account %s missing: %+v", want, list.Accounts)
		}
	}
	if !byName["SYS"].System || byName["APP"].Connections < 1 || !byName["APP"].JetStream {
		t.Fatalf("rows: %+v", byName)
	}

	app, err := m.Account("APP")
	if err != nil {
		t.Fatal(err)
	}
	user, err := m.Account("USER")
	if err != nil {
		t.Fatal(err)
	}
	var appD, userD struct {
		Exports []struct {
			Subject string `json:"subject"`
			Type    string `json:"type"`
		} `json:"exports"`
		Imports []struct {
			Subject string `json:"subject"`
			Account string `json:"account"`
			Type    string `json:"type"`
		} `json:"imports"`
		Mappings map[string]any `json:"mappings"`
	}
	if err := json.Unmarshal(app.Detail, &appD); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(user.Detail, &userD); err != nil {
		t.Fatal(err)
	}
	if len(appD.Exports) != 2 || appD.Mappings["orders.legacy.>"] == nil {
		t.Fatalf("APP exports/mappings: %+v", appD)
	}
	found := false
	for _, imp := range userD.Imports {
		found = found || (imp.Account == "APP" && imp.Subject == "events.>" && imp.Type == "stream")
	}
	if !found {
		t.Fatalf("USER should import events.> from APP: %+v", userD.Imports)
	}
	if _, err := m.Account("NOPE"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown account: %v", err)
	}
}

func TestAccountScopeSeesOnlyItsOwnAccount(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	m := monitorFor(t, c, "admin")
	// JetStream answers once the node has elected itself meta leader.
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		list, err := m.Accounts()
		if err == nil && len(list.Accounts) == 1 && list.Accounts[0].Name == "APP" && list.Source == Account && list.Accounts[0].JetStream {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("got %+v, %v", list, err)
		}
	}
	if _, err := m.Account("APP"); !errors.Is(err, ErrSystemOnly) {
		t.Fatalf("detail without system account: %v", err)
	}
}
