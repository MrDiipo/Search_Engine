package memory

import (
	"Web_Crawler/link_graph/graph"
	"gopkg.in/check.v1"
	"testing"
)

type InMemoryTestSuite struct {
	graph.SuiteBase
}

var _ = check.Suite(new(InMemoryTestSuite))

func (s *InMemoryTestSuite) setUpTest(c *check.C) {
	s.SetGraph(NewInMemoryGraph())
}

// Register our test-suite with go test.
func Test(t *testing.T) { check.TestingT(t) }
