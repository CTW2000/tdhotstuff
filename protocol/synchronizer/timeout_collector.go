package synchronizer

import (
	"slices"

	"github.com/relab/hotstuff"
	"github.com/relab/hotstuff/core"
)

type timeoutCollector struct {
	config   *core.RuntimeConfig
	timeouts map[hotstuff.View][]hotstuff.TimeoutMsg
}

func newTimeoutCollector(config *core.RuntimeConfig) *timeoutCollector {
	return &timeoutCollector{
		config: config,
	}
}

// add returns true if a quorum of timeouts has been collected for the view of given timeout message.
func (s *timeoutCollector) add(timeout hotstuff.TimeoutMsg) ([]hotstuff.TimeoutMsg, bool) {
	if s.timeouts == nil {
		s.timeouts = make(map[hotstuff.View][]hotstuff.TimeoutMsg)
	}

	view := timeout.View
	viewTimeouts := s.timeouts[view]

	// ignore this timeout if we already have a timeout from this replica in this view
	if slices.ContainsFunc(viewTimeouts, func(t hotstuff.TimeoutMsg) bool {
		return t.ID == timeout.ID
	}) {
		return nil, false
	}

	viewTimeouts = append(viewTimeouts, timeout)
	s.timeouts[view] = viewTimeouts

	if len(viewTimeouts) < s.config.QuorumSize() {
		return nil, false
	}

	timeoutList := slices.Clone(viewTimeouts)
	// remove timeouts for this view since we now have a quorum
	delete(s.timeouts, view)
	return timeoutList, true
}

// deleteOldViews removes all timeouts with a view lower than the current view.
// This is used to clean up timeouts that are no longer relevant, as they are from
// an already processed view.
func (s *timeoutCollector) deleteOldViews(currentView hotstuff.View) {
	for view := range s.timeouts {
		if view < currentView {
			delete(s.timeouts, view)
		}
	}
}
