package jobs

import (
	"context"
	"database/sql"
	"fmt"
)

type CleanUpCounts struct {
	RefreshTokensDeleted int32
	ActionTokensDeleted  int32
	APIKeysDeleted       int32
}

func CleanUpExpired(ctx context.Context, conn *sql.DB) (CleanUpCounts, error) {
	var counts CleanUpCounts

	row := conn.QueryRowContext(ctx, "select * from run_all_cleanups()")

	err := row.Scan(&counts.RefreshTokensDeleted, &counts.ActionTokensDeleted, &counts.APIKeysDeleted)
	if err != nil {
		return CleanUpCounts{}, fmt.Errorf("failure during execution of database cleanup function results; %w", err)
	}

	return counts, nil
}
