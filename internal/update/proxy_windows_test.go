package update

import (
	"net/url"
	"testing"
)

func TestWinProxyFor(t *testing.T) {
	u, _ := url.Parse("https://api.github.com/repos/x")
	if got := winProxyFor("proxy.example.com:8080", "", u); got == nil || got.Host != "proxy.example.com:8080" {
		t.Errorf("one proxy for all: %v", got)
	}
	if got := winProxyFor("http=a.example.com:80;https=b.example.com:443", "", u); got == nil || got.Host != "b.example.com:443" {
		t.Errorf("per scheme: %v", got)
	}
	if got := winProxyFor("proxy.example.com:8080", "*.github.com;<local>", u); got != nil {
		t.Errorf("a bypassed host goes direct: %v", got)
	}
	if got := winProxyFor("", "", u); got != nil {
		t.Errorf("no proxy: %v", got)
	}
	if got := pickProxyVals(true, proxyVals{server: "u"}, proxyVals{}, proxyVals{server: "p"}); got.server != "p" {
		t.Errorf("per machine falls back to the policy: %+v", got)
	}
	if got := pickProxyVals(false, proxyVals{server: "u"}, proxyVals{server: "m"}, proxyVals{}); got.server != "u" {
		t.Errorf("per user: %+v", got)
	}
}
