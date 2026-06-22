// SPDX-License-Identifier: MIT

package auth

import "github.com/jackc/pgx/v5/pgtype"

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
