// Copyright 2022 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package yaml

import (
	"encoding/json"
	"errors"
)

// Retry defines retry logic.
type Retry struct {
	Max  int           `json:"max,omitempty"`
	When Stringorslice `json:"when,omitempty"`
}

// UnmarshalJSON implements the unmarshal interface.
func (v *Retry) UnmarshalJSON(data []byte) error {
	var out1 int
	var out2 = struct {
		Max  int           `json:"max"`
		When Stringorslice `json:"when"`
	}{}

	if err := json.Unmarshal(data, &out1); err == nil {
		v.Max = out1
		return nil
	}

	if err := json.Unmarshal(data, &out2); err == nil {
		v.Max = out2.Max
		v.When = out2.When
		return nil
	}

	return errors.New("failed to unmarshal retry")
}

// Enum of retry when types
// https://docs.gitlab.com/ee/ci/yaml/#retrywhen
const (
	RetryAlways                 = "always"
	RetryUnknownFailure         = "unknown_failure"
	RetryScriptFailure          = "script_failure"
	RetryApiFailure             = "api_failure"
	RetrySturckOrTimeoutFailure = "stuck_or_timeout_failure"
	RetryRunnerSystemFailure    = "runner_system_failure"
	RetryRunnerUnsupported      = "runner_unsupported"
	RetryStaleSchedule          = "stale_schedule"
	RetryJobExecutionTimeout    = "job_execution_timeout"
	RetryArchivedFailure        = "archived_failure"
	RetrySchedulerFailure       = "scheduler_failure"
	RetryIntegrityFailure       = "data_integrity_failure"
)
