package service

import "testing"

func TestShouldClearStickySessionCustomFallsBackWhenNoSiteReason(t *testing.T) {
	clear, handled := shouldClearStickySessionCustom(&Account{Status: StatusActive, Schedulable: true}, "gpt-5.5")
	if clear || handled {
		t.Fatalf("healthy account should use official sticky-session fallback: clear=%v handled=%v", clear, handled)
	}
}

func TestShouldClearStickySessionCustomHandlesSiteSchedulingBlock(t *testing.T) {
	clear, handled := shouldClearStickySessionCustom(&Account{Status: StatusActive, Schedulable: false}, "gpt-5.5")
	if !clear || !handled {
		t.Fatalf("site scheduling block should be handled by overlay: clear=%v handled=%v", clear, handled)
	}
}
