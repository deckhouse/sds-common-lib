/*
Copyright 2026 Flant JSC

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

// Package slogh is a slog.Handler with the defaults the Deckhouse ecosystem
// wants, plus reconfiguration from a file at runtime.
//
// Deprecated: use the logger package instead.
//
// The storage modules are consolidating on one logger, built on
// github.com/deckhouse/deckhouse/pkg/log, so that a message looks the same
// whichever module emitted it and so that the slog linters see every call site.
// slogh is a second handler with its own configuration surface and its own level
// vocabulary, which is the thing that consolidation is meant to remove.
//
// The one feature it has that logger does not is reloading the level from a
// mounted file without restarting the pod. Where that matters, raise the level
// through the module's LOG_LEVEL and let the pod roll, or call
// logger.Logger.SetVerbosity, which changes the level in place on every logger
// derived from it.
//
// The package is kept compiling for the callers that have not moved yet
// (sds-replicated-volume's controller and agent, and the webhooks of
// snapshot-controller and storage-foundation); it takes no new features.
package slogh
