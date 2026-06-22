// SPDX-License-Identifier: MIT

package identity

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/entity"
	"github.com/smices/open-idb/internal/id"
)

// storeQueries defines the database operations used by Store.
// *generated.Queries satisfies this interface.
type storeQueries interface {
	GetDirectoryUserByExternalID(ctx context.Context, arg generated.GetDirectoryUserByExternalIDParams) (generated.DirectoryUser, error)
	GetManagedUserByBinding(ctx context.Context, arg generated.GetManagedUserByBindingParams) (generated.User, error)
	GetAccountBindingByProviderUID(ctx context.Context, arg generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error)
	ListAccountBindingsByUser(ctx context.Context, arg generated.ListAccountBindingsByUserParams) ([]generated.AccountBinding, error)
	GetAccountBindingByID(ctx context.Context, arg generated.GetAccountBindingByIDParams) (generated.AccountBinding, error)
	DeleteAccountBindingByID(ctx context.Context, arg DeleteAccountBindingByIDParams) error
	UpsertDirectoryUser(ctx context.Context, arg generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error)
	CreateAccountBinding(ctx context.Context, arg generated.CreateAccountBindingParams) (generated.AccountBinding, error)
	CreateManagedUser(ctx context.Context, arg generated.CreateManagedUserParams) (generated.User, error)
	GetUserByEntityAndID(ctx context.Context, arg generated.GetUserByEntityAndIDParams) (generated.User, error)
}

// Transactor begins database transactions.
// *pgxpool.Pool satisfies this interface.
type Transactor interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// ProvisionResult holds the outcome of ProvisionAndBind.
type ProvisionResult struct {
	ManagedUser ManagedUser
	Binding     AccountBinding
}

// ManagedUser is the domain representation of a managed (internal) user.
type ManagedUser struct {
	ID              UserID
	EntityID        entity.ID
	Username        string
	DisplayName     string
	Email           string
	Phone           string
	AvatarURL       string
	LifecycleStatus UserLifecycleStatus
	UserType        UserType
	PrimarySourceID SourceID
	Locale          string
}

// Store provides identity store operations.
type Store struct {
	q          storeQueries
	transactor Transactor
	txFactory  func(pgx.Tx) Store
}

// NewStore creates a Store backed by the given queries.
func NewStore(q *generated.Queries) Store {
	return Store{q: q}
}

// WithPool returns a copy of the Store with transaction support via the given pool.
func (s Store) WithPool(pool Transactor) Store {
	s.transactor = pool
	if s.txFactory == nil {
		s.txFactory = func(tx pgx.Tx) Store {
			return Store{q: generated.New(tx), transactor: s.transactor}
		}
	}
	return s
}

// GetDirectoryUser looks up a directory user by external ID.
func (s Store) GetDirectoryUser(ctx context.Context, entityID entity.ID, sourceID SourceID, externalUserID string) (DirectoryUser, error) {
	tid, err := parseEntityID(entityID)
	if err != nil {
		return DirectoryUser{}, fmt.Errorf("invalid entity_id: %w", err)
	}
	sid, err := parseSourceID(sourceID)
	if err != nil {
		return DirectoryUser{}, fmt.Errorf("invalid source_id: %w", err)
	}
	row, err := s.q.GetDirectoryUserByExternalID(ctx, generated.GetDirectoryUserByExternalIDParams{
		EntityID:       tid,
		SourceID:       sid,
		ExternalUserID: externalUserID,
	})
	if err != nil {
		return DirectoryUser{}, err
	}
	return toDomainDirectoryUser(row), nil
}

// GetManagedUserByBinding finds a managed user via account binding.
func (s Store) GetManagedUserByBinding(ctx context.Context, entityID entity.ID, sourceID SourceID, providerUID string) (ManagedUser, error) {
	tid, err := parseEntityID(entityID)
	if err != nil {
		return ManagedUser{}, fmt.Errorf("invalid entity_id: %w", err)
	}
	sid, err := parseSourceID(sourceID)
	if err != nil {
		return ManagedUser{}, fmt.Errorf("invalid source_id: %w", err)
	}
	row, err := s.q.GetManagedUserByBinding(ctx, generated.GetManagedUserByBindingParams{
		EntityID:    tid,
		SourceID:    sid,
		ProviderUid: providerUID,
	})
	if err != nil {
		return ManagedUser{}, err
	}
	return toDomainManagedUser(row), nil
}

// GetAccountBinding looks up a binding by provider UID.
func (s Store) GetAccountBinding(ctx context.Context, entityID entity.ID, sourceID SourceID, providerUID string) (AccountBinding, error) {
	tid, err := parseEntityID(entityID)
	if err != nil {
		return AccountBinding{}, fmt.Errorf("invalid entity_id: %w", err)
	}
	sid, err := parseSourceID(sourceID)
	if err != nil {
		return AccountBinding{}, fmt.Errorf("invalid source_id: %w", err)
	}
	row, err := s.q.GetAccountBindingByProviderUID(ctx, generated.GetAccountBindingByProviderUIDParams{
		EntityID:    tid,
		SourceID:    sid,
		ProviderUid: providerUID,
	})
	if err != nil {
		return AccountBinding{}, err
	}
	return toDomainAccountBinding(row), nil
}

// ListAccountBindings returns all bindings for a user.
func (s Store) ListAccountBindings(ctx context.Context, entityID entity.ID, userID UserID) ([]AccountBinding, error) {
	tid, err := parseEntityID(entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity_id: %w", err)
	}
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}
	rows, err := s.q.ListAccountBindingsByUser(ctx, generated.ListAccountBindingsByUserParams{
		EntityID: tid,
		UserID:   uid,
	})
	if err != nil {
		return nil, err
	}
	result := make([]AccountBinding, len(rows))
	for i, row := range rows {
		result[i] = toDomainAccountBinding(row)
	}
	return result, nil
}

// CreateAccountBinding creates a new account binding.
func (s Store) CreateAccountBinding(ctx context.Context, binding AccountBinding) (AccountBinding, error) {
	tid, err := parseEntityID(binding.EntityID)
	if err != nil {
		return AccountBinding{}, fmt.Errorf("invalid entity_id: %w", err)
	}
	uid, err := parseUserID(binding.UserID)
	if err != nil {
		return AccountBinding{}, fmt.Errorf("invalid user_id: %w", err)
	}
	sid, err := parseSourceID(binding.SourceID)
	if err != nil {
		return AccountBinding{}, fmt.Errorf("invalid source_id: %w", err)
	}
	did, err := parseDirectoryUserID(binding.DirectoryUserID)
	if err != nil {
		return AccountBinding{}, fmt.Errorf("invalid directory_user_id: %w", err)
	}
	row, err := s.q.CreateAccountBinding(ctx, generated.CreateAccountBindingParams{
		EntityID:        tid,
		UserID:          uid,
		SourceID:        sid,
		DirectoryUserID: did,
		ProviderUid:     binding.ProviderUID,
		ProviderUnionID: pgText(binding.ProviderUnionID),
		IsPrimary:       binding.IsPrimary,
	})
	if err != nil {
		return AccountBinding{}, err
	}
	return toDomainAccountBinding(row), nil
}

// DeleteAccountBinding removes a binding.
func (s Store) DeleteAccountBinding(ctx context.Context, entityID entity.ID, bindingID string) error {
	tid, err := parseEntityID(entityID)
	if err != nil {
		return fmt.Errorf("invalid entity_id: %w", err)
	}
	bid, err := parseULID(bindingID)
	if err != nil {
		return fmt.Errorf("invalid binding_id: %w", err)
	}
	return s.q.DeleteAccountBindingByID(ctx, DeleteAccountBindingByIDParams{
		EntityID: tid,
		ID:       bid,
	})
}

// UpsertDirectoryUser creates or updates a directory user.
func (s Store) UpsertDirectoryUser(ctx context.Context, user DirectoryUser) (DirectoryUser, error) {
	tid, err := parseEntityID(user.EntityID)
	if err != nil {
		return DirectoryUser{}, fmt.Errorf("invalid entity_id: %w", err)
	}
	sid, err := parseSourceID(user.SourceID)
	if err != nil {
		return DirectoryUser{}, fmt.Errorf("invalid source_id: %w", err)
	}
	row, err := s.q.UpsertDirectoryUser(ctx, generated.UpsertDirectoryUserParams{
		EntityID:        tid,
		SourceID:        sid,
		ExternalUserID:  user.ExternalUserID,
		ExternalUnionID: pgText(user.ExternalUnionID),
		ExternalOpenID:  pgText(user.ExternalOpenID),
		Name:            user.Name,
		Email:           pgText(user.Email),
		Phone:           pgText(user.Phone),
		AvatarUrl:       pgText(user.AvatarURL),
		Status:          string(user.Status),
		RawProfile:      []byte("{}"),
	})
	if err != nil {
		return DirectoryUser{}, err
	}
	return toDomainDirectoryUser(row), nil
}

// ProvisionAndBind transactionally provisions a managed user and creates a primary account binding.
// Requires WithPool to have been called; returns an error if no Transactor is configured.
func (s Store) ProvisionAndBind(ctx context.Context, dirUser DirectoryUser, policy ProvisionPolicy) (ProvisionResult, error) {
	if s.transactor == nil {
		return ProvisionResult{}, fmt.Errorf("no transactor configured: call WithPool before ProvisionAndBind")
	}

	tx, err := s.transactor.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	txStore := s.withTx(tx)

	draft := ProvisionManagedUser(dirUser, policy)
	if draft.Username == "" {
		return ProvisionResult{}, fmt.Errorf("provisioning disabled or directory user has no usable identifier")
	}

	entityULID, _ := parseEntityID(dirUser.EntityID)
	sourceULID, _ := parseSourceID(dirUser.SourceID)

	created, err := txStore.q.CreateManagedUser(ctx, generated.CreateManagedUserParams{
		EntityID:        entityULID,
		Username:        draft.Username,
		DisplayName:     draft.DisplayName,
		Email:           pgText(draft.Email),
		Phone:           pgText(draft.Phone),
		AvatarUrl:       pgText(draft.AvatarURL),
		LifecycleStatus: string(draft.LifecycleStatus),
		UserType:        string(draft.UserType),
		PrimarySourceID: pgText(sourceULID),
		Locale:          pgText(draft.Locale),
	})
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("create managed user: %w", err)
	}

	dirULID, _ := parseDirectoryUserID(dirUser.ID)

	binding, err := txStore.q.CreateAccountBinding(ctx, generated.CreateAccountBindingParams{
		EntityID:        created.EntityID,
		UserID:          created.ID,
		SourceID:        sourceULID,
		DirectoryUserID: dirULID,
		ProviderUid:     dirUser.ExternalUserID,
		ProviderUnionID: pgText(dirUser.ExternalUnionID),
		IsPrimary:       true,
	})
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("create account binding: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ProvisionResult{}, fmt.Errorf("commit: %w", err)
	}

	return ProvisionResult{
		ManagedUser: toDomainManagedUser(created),
		Binding:     toDomainAccountBinding(binding),
	}, nil
}

// --- Internal helpers ---

// DeleteAccountBindingByIDParams mirrors generated.DeleteAccountBindingByIDParams
// so the storeQueries interface can reference it directly.
type DeleteAccountBindingByIDParams = generated.DeleteAccountBindingByIDParams

func (s Store) withTx(tx pgx.Tx) Store {
	if s.txFactory != nil {
		return s.txFactory(tx)
	}
	return Store{q: generated.New(tx), transactor: s.transactor}
}

// --- ULID / type conversion helpers ---

func parseULID(s string) (string, error) {
	if err := id.ValidateULID(s); err != nil {
		return "", err
	}
	return s, nil
}

func parseEntityID(id entity.ID) (string, error) {
	return parseULID(string(id))
}

func parseSourceID(id SourceID) (string, error) {
	return parseULID(string(id))
}

func parseUserID(id UserID) (string, error) {
	return parseULID(string(id))
}

func parseDirectoryUserID(id DirectoryUserID) (string, error) {
	return parseULID(string(id))
}

func ulidToString(u string) string {
	return u
}

func textULIDToString(u pgtype.Text) string {
	if !u.Valid {
		return ""
	}
	return u.String
}

func pgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// --- Domain conversion helpers ---

func toDomainDirectoryUser(g generated.DirectoryUser) DirectoryUser {
	return DirectoryUser{
		ID:              DirectoryUserID(ulidToString(g.ID)),
		EntityID:        entity.ID(ulidToString(g.EntityID)),
		SourceID:        SourceID(ulidToString(g.SourceID)),
		ExternalUserID:  g.ExternalUserID,
		ExternalUnionID: g.ExternalUnionID.String,
		ExternalOpenID:  g.ExternalOpenID.String,
		Name:            g.Name,
		Email:           g.Email.String,
		Phone:           g.Phone.String,
		AvatarURL:       g.AvatarUrl.String,
		Status:          DirectoryUserStatus(g.Status),
	}
}

func toDomainManagedUser(g generated.User) ManagedUser {
	return ManagedUser{
		ID:              UserID(ulidToString(g.ID)),
		EntityID:        entity.ID(ulidToString(g.EntityID)),
		Username:        g.Username,
		DisplayName:     g.DisplayName,
		Email:           g.Email.String,
		Phone:           g.Phone.String,
		AvatarURL:       g.AvatarUrl.String,
		LifecycleStatus: UserLifecycleStatus(g.LifecycleStatus),
		UserType:        UserType(g.UserType),
		PrimarySourceID: SourceID(textULIDToString(g.PrimarySourceID)),
		Locale:          g.Locale.String,
	}
}

func toDomainAccountBinding(g generated.AccountBinding) AccountBinding {
	return AccountBinding{
		ID:              ulidToString(g.ID),
		EntityID:        entity.ID(ulidToString(g.EntityID)),
		UserID:          UserID(ulidToString(g.UserID)),
		SourceID:        SourceID(ulidToString(g.SourceID)),
		DirectoryUserID: DirectoryUserID(ulidToString(g.DirectoryUserID)),
		ProviderUID:     g.ProviderUid,
		ProviderUnionID: g.ProviderUnionID.String,
		IsPrimary:       g.IsPrimary,
	}
}
