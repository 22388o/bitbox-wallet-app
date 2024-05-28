// Copyright 2018 Shift Devices AG
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

package types

// Event instances are sent to the onEvent callback of the wallet.
type Event string

const (
	// EventStatusChanged is fired when the status changes. Check the status using Initialized().
	EventStatusChanged Event = "statusChanged"

	// EventSyncStarted is fired when syncing with the blockchain starts. This happens in the very
	// beginning for the initial sync, and repeatedly afterwards when the wallet is updated (new
	// transactions, confirmations, etc.).
	EventSyncStarted Event = "syncstarted"

	// EventSyncDone follows EventSyncStarted.
	EventSyncDone Event = "syncdone"

	// EventHeadersSynced is fired when the headers finished syncing.
	EventHeadersSynced Event = "headersSynced"
)
