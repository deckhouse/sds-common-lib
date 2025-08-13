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

package failer

import "github.com/deckhouse/sds-common-lib/fs"

// Failure-injection interface
type Failer interface {
	// Checks if the called operation should fail
	// `mockFs` - the MockFs object
	// Arguments helping to make a decision:
	// `op`     - called operation
	// `self`   - object which method is called (e.g. Fd). Can be nil (e.g. for methods of `Fs`)
	// `args`   - the arguments of the operation
	ShouldFail(os fs.OS, op fs.Op, self any, args ...any) error
}
