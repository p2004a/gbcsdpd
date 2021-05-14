// Copyright 2021 Google LLC
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

package backoff

import (
	"math"
	"math/rand"
	"time"
)

// Exponential computes how much time to wait before retrying.
func Exponential(retryNum int, baseDelay time.Duration, maxDelay time.Duration, factor float64) time.Duration {
	if retryNum == 0 {
		return 0
	}
	backoff := float64(baseDelay) * math.Pow(factor, float64(retryNum))
	backoff -= backoff * 0.5 * rand.Float64() // add jitter
	return time.Duration(math.Min(float64(maxDelay), backoff))
}
