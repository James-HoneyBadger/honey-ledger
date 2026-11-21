// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package middlewares

import (
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/settings"
	"github.com/James-HoneyBadger/honey-ledger/pkg/utils"
)

// MCPServerIpLimit limits access to the MCP server based on IP address.
func MCPServerIpLimit(config *settings.Config) core.MiddlewareHandlerFunc {
	return func(c *core.WebContext) {
		if len(config.MCPAllowedRemoteIPs) < 1 {
			c.Next()
			return
		}

		for i := 0; i < len(config.MCPAllowedRemoteIPs); i++ {
			if config.MCPAllowedRemoteIPs[i].Match(c.ClientIP()) {
				c.Next()
				return
			}
		}

		utils.PrintJsonErrorResult(c, errs.ErrIPForbidden)
	}
}
