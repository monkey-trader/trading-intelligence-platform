package position

import (
	"context"
	"fmt"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// SetSwingLongUseCase markiert eine Position als Swing Long (idempotent, ohne Existenzprüfung).
type SetSwingLongUseCase struct {
	flags interfaces.SwingLongRepository
	now   func() int64
}

// NewSetSwingLongUseCase baut den Use Case.
func NewSetSwingLongUseCase(flags interfaces.SwingLongRepository, now func() int64) *SetSwingLongUseCase {
	return &SetSwingLongUseCase{flags: flags, now: now}
}

// Execute setzt das Flag mit now() als setAt.
func (useCase *SetSwingLongUseCase) Execute(ctx context.Context, positionID string) error {
	if err := useCase.flags.Set(ctx, positionID, useCase.now()); err != nil {
		return fmt.Errorf("swing long setzen: %w", err)
	}
	return nil
}

// UnsetSwingLongUseCase entfernt das Flag (idempotent).
type UnsetSwingLongUseCase struct {
	flags interfaces.SwingLongRepository
}

// NewUnsetSwingLongUseCase baut den Use Case.
func NewUnsetSwingLongUseCase(flags interfaces.SwingLongRepository) *UnsetSwingLongUseCase {
	return &UnsetSwingLongUseCase{flags: flags}
}

// Execute entfernt das Flag.
func (useCase *UnsetSwingLongUseCase) Execute(ctx context.Context, positionID string) error {
	if err := useCase.flags.Unset(ctx, positionID); err != nil {
		return fmt.Errorf("swing long entfernen: %w", err)
	}
	return nil
}

// readSwingLongIDs liest die markierten IDs als Menge; ohne Repository ist sie leer.
func readSwingLongIDs(ctx context.Context, flags interfaces.SwingLongRepository) (map[string]bool, error) {
	if flags == nil {
		return nil, nil
	}
	ids, err := flags.ListIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("swing-long-flags lesen: %w", err)
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}
