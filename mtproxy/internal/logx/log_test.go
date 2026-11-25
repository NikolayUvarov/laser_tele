package logx

import "testing"

func TestCensor(t *testing.T) {
	cases := map[string]string{
		"connect kws2.web.telegram.org ok": "connect kws2.web.telegram.org ok",
		"worker my.account.workers.dev":    "worker m*.acc****.wor****.dev",
		"ip 149.154.167.220:443":           "ip 149.154.167.220:443",
		"file laser-mtproxy.log":           "file laser-mtproxy.log",
	}
	for in, want := range cases {
		if got := Censor(in); got != want {
			t.Errorf("Censor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRing(t *testing.T) {
	Infof("first")
	entries := Since(0, 0)
	last := entries[len(entries)-1]
	if last.Msg != "first" {
		t.Fatal(last)
	}
	Infof("second")
	if got := Since(last.Seq, 0); len(got) != 1 || got[0].Msg != "second" {
		t.Fatal(got)
	}
	Debugf("hidden")
	if got := Since(last.Seq, 0); len(got) != 1 {
		t.Fatal("debug written on info level")
	}
}
