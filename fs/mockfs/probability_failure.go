/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mockfs

import (
	"fmt"
	"math/rand"
)

// Generates random failures with the given probability
type ProbabilityFailer struct {
	probability float64
	rnd         *rand.Rand
}

// Creates a new ProbabilityFailer with the given probability
func NewProbabilityFailer(seed int64, prob float64) *ProbabilityFailer {
	prob = max(min(prob, 1.0), 0.0)

	return &ProbabilityFailer{
		probability: prob,
		rnd:         rand.New(rand.NewSource(seed)),
	}
}

func (pf *ProbabilityFailer) ShouldFail(_ *MockFS, op string, _ any, _ ...any) error {
	if pf.probability <= 0 {
		return nil
	}

	if pf.probability >= 1 || pf.rnd.Float64() < pf.probability {
		return fmt.Errorf("probability fail: op %s", op)
	}
	return nil
}
