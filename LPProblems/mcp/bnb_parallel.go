package mcp

import "sync"

// runParallelBranches processes two branches (x_v=0 и x_v=1) at the same time.
// Returns done=true, if both branches were fully complete.
func (s *bnbSolver) runParallelBranches(
	branchVar int,
	fixed []nodeState,
	colLower, colUpper []float64,
	depth int,
) (bool, error) {
	if !s.tryAcquireParallelSlot() {
		return false, nil
	}

	leftFixed := append([]nodeState(nil), fixed...)
	leftLower := append([]float64(nil), colLower...)
	leftUpper := append([]float64(nil), colUpper...)
	leftFixed[branchVar] = nodeForceZero
	leftLower[branchVar] = 0
	leftUpper[branchVar] = 0

	rightFixed := append([]nodeState(nil), fixed...)
	rightLower := append([]float64(nil), colLower...)
	rightUpper := append([]float64(nil), colUpper...)
	rightFixed[branchVar] = nodeForceOne
	rightLower[branchVar] = 1
	rightUpper[branchVar] = 1

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer s.releaseParallelSlot()
		if err := s.branch(leftFixed, leftLower, leftUpper, depth+1); err != nil {
			errCh <- err
		}
	}()
	if err := s.branch(rightFixed, rightLower, rightUpper, depth+1); err != nil {
		errCh <- err
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return true, err
		}
	}
	return true, nil
}

func (s *bnbSolver) tryAcquireParallelSlot() bool {
	select {
	case s.parallelSem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *bnbSolver) releaseParallelSlot() {
	<-s.parallelSem
}
