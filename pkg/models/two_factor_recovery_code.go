// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package models

// TwoFactorRecoveryCode represents user 2fa recovery codes stored in database
type TwoFactorRecoveryCode struct {
	Uid             int64  `xorm:"PK"`
	RecoveryCode    string `xorm:"VARCHAR(64) PK"`
	Used            bool   `xorm:"NOT NULL"`
	CreatedUnixTime int64
	UsedUnixTime    int64
}

// TwoFactorRecoveryCodeLoginRequest represents all parameters of 2fa login request via recovery code
type TwoFactorRecoveryCodeLoginRequest struct {
	RecoveryCode string `json:"recoveryCode" binding:"required,notBlank,len=11"`
}
