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

package fake

import (
	"fmt"
	"reflect"
)

func tryCastAndSetArgument[T comparable](ptr *T, arg any, found *bool, makeError func() error) error {
	var zero T // nil for interface
	if arg, ok := arg.(T); ok {
		if *ptr != zero && *ptr != arg {
			return fmt.Errorf("%v already set: %w", reflect.TypeFor[T](), makeError())
		}
		*ptr = arg
		*found = true
	}
	return nil
}
