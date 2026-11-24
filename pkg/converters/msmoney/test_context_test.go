// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import "context"

// testContext stands in for the application context the applications pass.
type testContext struct {
	context.Context
}

func (c *testContext) GetContextId() string {
	return "test"
}

func (c *testContext) GetClientLocale() string {
	return ""
}

func newTestContext() Context {
	return &testContext{Context: context.Background()}
}
