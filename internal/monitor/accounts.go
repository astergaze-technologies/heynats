package monitor

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// AccountRow is one account summed over every server. Limits are -1 when
// unknown or unlimited.
type AccountRow struct {
	Name          string `json:"name"`
	System        bool   `json:"system"`
	Connections   int    `json:"connections"`
	LeafNodes     int    `json:"leafnodes"`
	Subscriptions uint32 `json:"subscriptions"`
	MsgsSent      int64  `json:"msgs_sent"`
	MsgsReceived  int64  `json:"msgs_received"`
	BytesSent     int64  `json:"bytes_sent"`
	BytesReceived int64  `json:"bytes_received"`
	SlowConsumers int64  `json:"slow_consumers"`
	JetStream     bool   `json:"jetstream"`
	JSMemory      uint64 `json:"js_memory"`
	JSStorage     uint64 `json:"js_storage"`
	JSMaxMemory   int64  `json:"js_max_memory"`
	JSMaxStorage  int64  `json:"js_max_storage"`
}

type AccountList struct {
	Source   Source       `json:"source"`
	Partial  bool         `json:"partial"`
	Servers  int          `json:"servers"`
	Expected int          `json:"expected"`
	Accounts []AccountRow `json:"accounts"`
}

type accountStat struct {
	Account  string `json:"acc"`
	Conns    int    `json:"conns"`
	Leafs    int    `json:"leafnodes"`
	Subs     uint32 `json:"num_subscriptions"`
	Slow     int64  `json:"slow_consumers"`
	Sent     struct{ Msgs, Bytes int64 }
	Received struct{ Msgs, Bytes int64 }
}

// addStats sums account STATZ replies into rows, creating rows as needed.
func addStats(rows map[string]*AccountRow, res *Result) {
	for _, r := range res.Replies {
		var d struct {
			Accounts []accountStat `json:"account_statz"`
		}
		if r.Error != nil || json.Unmarshal(r.Data, &d) != nil {
			continue
		}
		for _, a := range d.Accounts {
			row := rows[a.Account]
			if row == nil {
				row = &AccountRow{Name: a.Account, JSMaxMemory: -1, JSMaxStorage: -1}
				rows[a.Account] = row
			}
			row.Connections += a.Conns
			row.LeafNodes += a.Leafs
			row.Subscriptions += a.Subs
			row.SlowConsumers += a.Slow
			row.MsgsSent += a.Sent.Msgs
			row.BytesSent += a.Sent.Bytes
			row.MsgsReceived += a.Received.Msgs
			row.BytesReceived += a.Received.Bytes
		}
	}
}

func (m *Monitor) Accounts() (*AccountList, error) {
	if m.Source() != System {
		return m.ownAccount()
	}
	type out struct {
		res *Result
		err error
	}
	statzC, jszC, accountzC := make(chan out, 1), make(chan out, 1), make(chan out, 1)
	go func() {
		r, err := m.gather("$SYS.REQ.ACCOUNT.PING.STATZ", []byte(`{"include_unused":true}`))
		statzC <- out{r, err}
	}()
	go func() { r, err := m.PingWith("JSZ", map[string]bool{"accounts": true}); jszC <- out{r, err} }()
	go func() { r, err := m.Ping("ACCOUNTZ"); accountzC <- out{r, err} }()
	statz, jsz, accountz := <-statzC, <-jszC, <-accountzC
	if statz.err != nil {
		return nil, statz.err
	}

	rows := map[string]*AccountRow{}
	addStats(rows, statz.res)
	if accountz.err == nil && len(accountz.res.Replies) > 0 {
		var d struct {
			System   string   `json:"system_account"`
			Accounts []string `json:"accounts"`
		}
		if json.Unmarshal(accountz.res.Replies[0].Data, &d) == nil {
			for _, name := range d.Accounts {
				if rows[name] == nil {
					rows[name] = &AccountRow{Name: name, JSMaxMemory: -1, JSMaxStorage: -1}
				}
			}
			if row := rows[d.System]; row != nil {
				row.System = true
			}
		}
	}
	if jsz.err == nil {
		for _, r := range jsz.res.Replies {
			var d struct {
				Accounts []struct {
					Name    string `json:"name"`
					Memory  uint64 `json:"memory"`
					Storage uint64 `json:"storage"`
				} `json:"account_details"`
			}
			if r.Error != nil || json.Unmarshal(r.Data, &d) != nil {
				continue
			}
			for _, a := range d.Accounts {
				if row := rows[a.Name]; row != nil {
					row.JetStream = true
					row.JSMemory += a.Memory
					row.JSStorage += a.Storage
				}
			}
		}
	}

	list := &AccountList{Source: System, Partial: statz.res.Partial, Servers: len(statz.res.Replies), Expected: statz.res.Expected}
	for _, row := range rows {
		list.Accounts = append(list.Accounts, *row)
	}
	sort.Slice(list.Accounts, func(i, j int) bool { return list.Accounts[i].Name < list.Accounts[j].Name })
	return list, nil
}

// ownAccount is the account-scope view: the user's own account only.
func (m *Monitor) ownAccount() (*AccountList, error) {
	statz, err := m.Ping("STATZ")
	if err != nil {
		return nil, err
	}
	rows := map[string]*AccountRow{}
	addStats(rows, statz)
	list := &AccountList{Source: Account, Partial: statz.Partial, Servers: len(statz.Replies), Expected: statz.Expected}
	for _, row := range rows {
		if js, err := m.accountJetStream(); err == nil {
			row.JetStream = true
			row.JSMemory, row.JSStorage = js.Memory, js.Storage
			row.JSMaxMemory, row.JSMaxStorage = js.MaxMemory, js.MaxStorage
		}
		list.Accounts = append(list.Accounts, *row)
	}
	return list, nil
}

// AccountDetail is ACCOUNTZ for one account: imports, exports, mappings and,
// in operator mode, the decoded JWT. Kept raw beyond what the UI lists.
type AccountDetail struct {
	Name      string          `json:"name"`
	UpdatedAt time.Time       `json:"update_time,omitzero"`
	Detail    json.RawMessage `json:"detail"`
}

func (m *Monitor) Account(name string) (*AccountDetail, error) {
	if m.Source() != System {
		return nil, ErrSystemOnly
	}
	res, err := m.PingWith("ACCOUNTZ", map[string]string{"account": name})
	if err != nil {
		return nil, err
	}
	// Account config is the same on every server; the first full answer will do.
	for _, r := range res.Replies {
		var d struct {
			Detail json.RawMessage `json:"account_detail"`
		}
		if r.Error != nil || json.Unmarshal(r.Data, &d) != nil || len(d.Detail) == 0 {
			continue
		}
		var head struct {
			Name    string    `json:"account_name"`
			Updated time.Time `json:"update_time"`
		}
		_ = json.Unmarshal(d.Detail, &head)
		return &AccountDetail{Name: head.Name, UpdatedAt: head.Updated, Detail: d.Detail}, nil
	}
	return nil, fmt.Errorf("%w: no account %q", ErrNotFound, name)
}
