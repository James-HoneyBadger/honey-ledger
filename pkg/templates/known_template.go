// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package templates

type KnownTemplate string

// Known templates
const (
	TEMPLATE_VERIFY_EMAIL   KnownTemplate = "email/verify_email"
	TEMPLATE_PASSWORD_RESET KnownTemplate = "email/password_reset"
)
