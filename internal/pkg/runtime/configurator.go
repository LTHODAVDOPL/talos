/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package runtime

import (
	"github.com/talos-systems/talos/pkg/config/cluster"
	"github.com/talos-systems/talos/pkg/config/machine"
)

// Configurator defines the configuration interface.
type Configurator interface {
	Version() string
	Debug() bool
	Machine() machine.Machine
	Cluster() cluster.Cluster
	Validate(Mode) error
	String() (string, error)
}
