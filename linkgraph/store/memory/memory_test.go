package memory

import (
	"Search_Engine/linkgraph/graph"
	_ "gopkg.in/check.v1"
	gc "gopkg.in/check.v1"
	"testing"
)

type InMemoryTestSuite struct {
	graph.SuiteBase
}

var _ = gc.Suite(new(InMemoryGraphTestSuite))

func Test(t *testing.T) { gc.TestingT(t) }

type InMemoryGraphTestSuite struct {
	graphtest.SuiteBase
}

func (s *InMemoryGraphTestSuite) SetUpTest(c *gc.C) {
	s.SetGraph(NewInMemoryGraph())
}
