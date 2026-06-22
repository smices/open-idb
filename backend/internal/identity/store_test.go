// SPDX-License-Identifier: MIT

package identity

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/entity"
	"github.com/smices/open-idb/internal/id"
)

// --- Mock implementations ---

type mockStoreQueries struct {
	getDirUserFn    func(context.Context, generated.GetDirectoryUserByExternalIDParams) (generated.DirectoryUser, error)
	getManagedFn    func(context.Context, generated.GetManagedUserByBindingParams) (generated.User, error)
	getBindingFn    func(context.Context, generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error)
	listBindingsFn  func(context.Context, generated.ListAccountBindingsByUserParams) ([]generated.AccountBinding, error)
	getBindingIDFn  func(context.Context, generated.GetAccountBindingByIDParams) (generated.AccountBinding, error)
	deleteBindingFn func(context.Context, generated.DeleteAccountBindingByIDParams) error
	upsertDirUserFn func(context.Context, generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error)
	createBindingFn func(context.Context, generated.CreateAccountBindingParams) (generated.AccountBinding, error)
	createUserFn    func(context.Context, generated.CreateManagedUserParams) (generated.User, error)
	getUserFn       func(context.Context, generated.GetUserByEntityAndIDParams) (generated.User, error)
}

func (m *mockStoreQueries) GetDirectoryUserByExternalID(ctx context.Context, arg generated.GetDirectoryUserByExternalIDParams) (generated.DirectoryUser, error) {
	return m.getDirUserFn(ctx, arg)
}
func (m *mockStoreQueries) GetManagedUserByBinding(ctx context.Context, arg generated.GetManagedUserByBindingParams) (generated.User, error) {
	return m.getManagedFn(ctx, arg)
}
func (m *mockStoreQueries) GetAccountBindingByProviderUID(ctx context.Context, arg generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
	return m.getBindingFn(ctx, arg)
}
func (m *mockStoreQueries) ListAccountBindingsByUser(ctx context.Context, arg generated.ListAccountBindingsByUserParams) ([]generated.AccountBinding, error) {
	return m.listBindingsFn(ctx, arg)
}
func (m *mockStoreQueries) GetAccountBindingByID(ctx context.Context, arg generated.GetAccountBindingByIDParams) (generated.AccountBinding, error) {
	return m.getBindingIDFn(ctx, arg)
}
func (m *mockStoreQueries) DeleteAccountBindingByID(ctx context.Context, arg generated.DeleteAccountBindingByIDParams) error {
	return m.deleteBindingFn(ctx, arg)
}
func (m *mockStoreQueries) UpsertDirectoryUser(ctx context.Context, arg generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
	return m.upsertDirUserFn(ctx, arg)
}
func (m *mockStoreQueries) CreateAccountBinding(ctx context.Context, arg generated.CreateAccountBindingParams) (generated.AccountBinding, error) {
	return m.createBindingFn(ctx, arg)
}
func (m *mockStoreQueries) CreateManagedUser(ctx context.Context, arg generated.CreateManagedUserParams) (generated.User, error) {
	return m.createUserFn(ctx, arg)
}
func (m *mockStoreQueries) GetUserByEntityAndID(ctx context.Context, arg generated.GetUserByEntityAndIDParams) (generated.User, error) {
	return m.getUserFn(ctx, arg)
}

type mockTx struct {
	committed  bool
	rolledBack bool
}

func (m *mockTx) Begin(ctx context.Context) (pgx.Tx, error)                 { return nil, nil }
func (m *mockTx) BeginFunc(ctx context.Context, f func(pgx.Tx) error) error { return nil }
func (m *mockTx) Commit(ctx context.Context) error                          { m.committed = true; return nil }
func (m *mockTx) Rollback(ctx context.Context) error                        { m.rolledBack = true; return nil }
func (m *mockTx) Conn() *pgx.Conn                                           { return nil }
func (m *mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (m *mockTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (m *mockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *mockTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *mockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

type mockTransactor struct {
	tx  *mockTx
	err error
}

func (m *mockTransactor) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return m.tx, m.err
}

// --- Test helpers ---

var (
	testEntityULID = mustTestULID("01HZZZZZZZ0000000000000001")
	testSourceULID = mustTestULID("01HZZZZZZZ0000000000000002")
	testUserULID   = mustTestULID("01HZZZZZZZ0000000000000003")
	testDirULID    = mustTestULID("01HZZZZZZZ0000000000000004")
	testBindULID   = mustTestULID("01HZZZZZZZ0000000000000005")

	testEntityID = entity.ID(testEntityULID)
	testSourceID = SourceID(testSourceULID)
	testUserID   = UserID(testUserULID)
	testDirID    = DirectoryUserID(testDirULID)
)

func mustTestULID(s string) string {
	if err := id.ValidateULID(s); err != nil {
		panic(err)
	}
	return s
}

func testGenDirectoryUser() generated.DirectoryUser {
	return generated.DirectoryUser{
		ID:              testDirULID,
		EntityID:        testEntityULID,
		SourceID:        testSourceULID,
		ExternalUserID:  "ext_001",
		ExternalUnionID: pgtype.Text{String: "union_1", Valid: true},
		ExternalOpenID:  pgtype.Text{String: "open_1", Valid: true},
		Name:            "Test User",
		Email:           pgtype.Text{String: "test@example.com", Valid: true},
		Phone:           pgtype.Text{String: "13800000000", Valid: true},
		AvatarUrl:       pgtype.Text{String: "https://example.com/avatar.png", Valid: true},
		Status:          "active",
	}
}

func testGenAccountBinding() generated.AccountBinding {
	return generated.AccountBinding{
		ID:              testBindULID,
		EntityID:        testEntityULID,
		UserID:          testUserULID,
		SourceID:        testSourceULID,
		DirectoryUserID: testDirULID,
		ProviderUid:     "ext_001",
		ProviderUnionID: pgtype.Text{String: "union_1", Valid: true},
		IsPrimary:       true,
	}
}

func testGenUser() generated.User {
	return generated.User{
		ID:              testUserULID,
		EntityID:        testEntityULID,
		Username:        "test@example.com",
		DisplayName:     "Test User",
		Email:           pgtype.Text{String: "test@example.com", Valid: true},
		Phone:           pgtype.Text{String: "13800000000", Valid: true},
		AvatarUrl:       pgtype.Text{String: "https://example.com/avatar.png", Valid: true},
		LifecycleStatus: "active",
		UserType:        "employee",
		PrimarySourceID: pgtype.Text{String: testSourceULID, Valid: true},
		Locale:          pgtype.Text{String: "zh-CN", Valid: true},
	}
}

func testDomainDirectoryUser() DirectoryUser {
	return DirectoryUser{
		ID:              testDirID,
		EntityID:        testEntityID,
		SourceID:        testSourceID,
		ExternalUserID:  "ext_001",
		ExternalUnionID: "union_1",
		ExternalOpenID:  "open_1",
		Name:            "Test User",
		Email:           "test@example.com",
		Phone:           "13800000000",
		AvatarURL:       "https://example.com/avatar.png",
		Status:          DirectoryUserStatusActive,
	}
}

// --- NewStore test ---

func TestNewStore(t *testing.T) {
	store := NewStore(&generated.Queries{})
	if store.q == nil {
		t.Fatal("expected queries")
	}
}

// --- GetDirectoryUser tests ---

func TestGetDirectoryUser(t *testing.T) {
	q := &mockStoreQueries{
		getDirUserFn: func(_ context.Context, arg generated.GetDirectoryUserByExternalIDParams) (generated.DirectoryUser, error) {
			if arg.ExternalUserID != "ext_001" {
				t.Fatalf("ExternalUserID = %q, want ext_001", arg.ExternalUserID)
			}
			return testGenDirectoryUser(), nil
		},
	}
	store := Store{q: q}

	got, err := store.GetDirectoryUser(context.Background(), testEntityID, testSourceID, "ext_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := testDomainDirectoryUser()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGetDirectoryUserNotFound(t *testing.T) {
	q := &mockStoreQueries{
		getDirUserFn: func(_ context.Context, _ generated.GetDirectoryUserByExternalIDParams) (generated.DirectoryUser, error) {
			return generated.DirectoryUser{}, pgx.ErrNoRows
		},
	}
	store := Store{q: q}

	_, err := store.GetDirectoryUser(context.Background(), testEntityID, testSourceID, "nonexistent")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}
}

func TestGetDirectoryUserInvalidEntityID(t *testing.T) {
	store := Store{q: &mockStoreQueries{}}
	_, err := store.GetDirectoryUser(context.Background(), entity.ID("not-a-ulid"), testSourceID, "ext_001")
	if err == nil {
		t.Fatal("expected error for invalid entity ID")
	}
}

// --- GetManagedUserByBinding tests ---

func TestGetManagedUserByBinding(t *testing.T) {
	q := &mockStoreQueries{
		getManagedFn: func(_ context.Context, arg generated.GetManagedUserByBindingParams) (generated.User, error) {
			if arg.ProviderUid != "ext_001" {
				t.Fatalf("ProviderUid = %q", arg.ProviderUid)
			}
			return testGenUser(), nil
		},
	}
	store := Store{q: q}

	got, err := store.GetManagedUserByBinding(context.Background(), testEntityID, testSourceID, "ext_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DisplayName != "Test User" {
		t.Fatalf("DisplayName = %q", got.DisplayName)
	}
	if got.LifecycleStatus != UserLifecycleActive {
		t.Fatalf("LifecycleStatus = %q", got.LifecycleStatus)
	}
	if got.UserType != UserTypeEmployee {
		t.Fatalf("UserType = %q", got.UserType)
	}
	if got.Locale != "zh-CN" {
		t.Fatalf("Locale = %q", got.Locale)
	}
}

// --- GetAccountBinding tests ---

func TestGetAccountBinding(t *testing.T) {
	q := &mockStoreQueries{
		getBindingFn: func(_ context.Context, arg generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			if arg.ProviderUid != "ext_001" {
				t.Fatalf("ProviderUid = %q", arg.ProviderUid)
			}
			return testGenAccountBinding(), nil
		},
	}
	store := Store{q: q}

	got, err := store.GetAccountBinding(context.Background(), testEntityID, testSourceID, "ext_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ProviderUID != "ext_001" {
		t.Fatalf("ProviderUID = %q", got.ProviderUID)
	}
	if !got.IsPrimary {
		t.Fatal("expected IsPrimary = true")
	}
}

// --- ListAccountBindings tests ---

func TestListAccountBindings(t *testing.T) {
	q := &mockStoreQueries{
		listBindingsFn: func(_ context.Context, arg generated.ListAccountBindingsByUserParams) ([]generated.AccountBinding, error) {
			return []generated.AccountBinding{testGenAccountBinding()}, nil
		},
	}
	store := Store{q: q}

	got, err := store.ListAccountBindings(context.Background(), testEntityID, testUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ProviderUID != "ext_001" {
		t.Fatalf("ProviderUID = %q", got[0].ProviderUID)
	}
}

func TestListAccountBindingsEmpty(t *testing.T) {
	q := &mockStoreQueries{
		listBindingsFn: func(_ context.Context, _ generated.ListAccountBindingsByUserParams) ([]generated.AccountBinding, error) {
			return nil, nil
		},
	}
	store := Store{q: q}

	got, err := store.ListAccountBindings(context.Background(), testEntityID, testUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

// --- CreateAccountBinding tests ---

func TestCreateAccountBinding(t *testing.T) {
	var gotParams generated.CreateAccountBindingParams

	q := &mockStoreQueries{
		createBindingFn: func(_ context.Context, arg generated.CreateAccountBindingParams) (generated.AccountBinding, error) {
			gotParams = arg
			return testGenAccountBinding(), nil
		},
	}
	store := Store{q: q}

	input := AccountBinding{
		EntityID:        testEntityID,
		UserID:          testUserID,
		SourceID:        testSourceID,
		DirectoryUserID: testDirID,
		ProviderUID:     "ext_001",
		ProviderUnionID: "union_1",
		IsPrimary:       true,
	}

	got, err := store.CreateAccountBinding(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ProviderUID != "ext_001" {
		t.Fatalf("ProviderUID = %q", got.ProviderUID)
	}
	if !gotParams.IsPrimary {
		t.Fatal("expected IsPrimary in params")
	}
	if !gotParams.ProviderUnionID.Valid || gotParams.ProviderUnionID.String != "union_1" {
		t.Fatalf("ProviderUnionID = %v", gotParams.ProviderUnionID)
	}
}

// --- DeleteAccountBinding tests ---

func TestDeleteAccountBinding(t *testing.T) {
	var gotParams generated.DeleteAccountBindingByIDParams

	q := &mockStoreQueries{
		deleteBindingFn: func(_ context.Context, arg generated.DeleteAccountBindingByIDParams) error {
			gotParams = arg
			return nil
		},
	}
	store := Store{q: q}

	err := store.DeleteAccountBinding(context.Background(), testEntityID, testBindULID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.ID != testBindULID {
		t.Fatalf("ID mismatch")
	}
}

func TestDeleteAccountBindingInvalidID(t *testing.T) {
	store := Store{q: &mockStoreQueries{}}
	err := store.DeleteAccountBinding(context.Background(), testEntityID, "not-a-ulid")
	if err == nil {
		t.Fatal("expected error for invalid binding ID")
	}
}

// --- UpsertDirectoryUser tests ---

func TestUpsertDirectoryUser(t *testing.T) {
	var gotParams generated.UpsertDirectoryUserParams

	q := &mockStoreQueries{
		upsertDirUserFn: func(_ context.Context, arg generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			gotParams = arg
			return testGenDirectoryUser(), nil
		},
	}
	store := Store{q: q}

	input := DirectoryUser{
		EntityID:        testEntityID,
		SourceID:        testSourceID,
		ExternalUserID:  "ext_001",
		ExternalUnionID: "union_1",
		ExternalOpenID:  "open_1",
		Name:            "Test User",
		Email:           "test@example.com",
		Phone:           "13800000000",
		AvatarURL:       "https://example.com/avatar.png",
		Status:          DirectoryUserStatusActive,
	}

	got, err := store.UpsertDirectoryUser(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ExternalUserID != "ext_001" {
		t.Fatalf("ExternalUserID = %q", got.ExternalUserID)
	}
	if got.Name != "Test User" {
		t.Fatalf("Name = %q", got.Name)
	}
	if got.Status != DirectoryUserStatusActive {
		t.Fatalf("Status = %q", got.Status)
	}
	if gotParams.Status != "active" {
		t.Fatalf("params Status = %q", gotParams.Status)
	}
	if !gotParams.Email.Valid || gotParams.Email.String != "test@example.com" {
		t.Fatalf("params Email = %v", gotParams.Email)
	}
}

// --- ProvisionAndBind tests ---

func TestProvisionAndBindReturnsErrorWithoutPool(t *testing.T) {
	store := NewStore(nil)
	_, err := store.ProvisionAndBind(context.Background(), testDomainDirectoryUser(), ProvisionPolicy{
		AutoCreateManagedUsers: true,
		DefaultLifecycleStatus: UserLifecycleActive,
	})
	if err == nil {
		t.Fatal("expected error without pool")
	}
	if !strings.Contains(err.Error(), "transactor") {
		t.Fatalf("error = %q, want transactor error", err.Error())
	}
}

func TestProvisionAndBindTransactorError(t *testing.T) {
	store := Store{
		q: &mockStoreQueries{},
		transactor: &mockTransactor{
			err: errors.New("connection refused"),
		},
	}
	_, err := store.ProvisionAndBind(context.Background(), testDomainDirectoryUser(), ProvisionPolicy{
		AutoCreateManagedUsers: true,
		DefaultLifecycleStatus: UserLifecycleActive,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestProvisionAndBindSuccess(t *testing.T) {
	tx := &mockTx{}
	var gotCreateUserParams generated.CreateManagedUserParams
	var gotCreateBindingParams generated.CreateAccountBindingParams

	txQ := &mockStoreQueries{
		createUserFn: func(_ context.Context, arg generated.CreateManagedUserParams) (generated.User, error) {
			gotCreateUserParams = arg
			return testGenUser(), nil
		},
		createBindingFn: func(_ context.Context, arg generated.CreateAccountBindingParams) (generated.AccountBinding, error) {
			gotCreateBindingParams = arg
			return testGenAccountBinding(), nil
		},
	}

	store := Store{
		q:          &mockStoreQueries{},
		transactor: &mockTransactor{tx: tx},
		txFactory:  func(pgx.Tx) Store { return Store{q: txQ} },
	}

	dirUser := testDomainDirectoryUser()
	policy := ProvisionPolicy{
		AutoCreateManagedUsers: true,
		DefaultLifecycleStatus: UserLifecycleActive,
		DefaultLocale:          "zh-CN",
	}

	result, err := store.ProvisionAndBind(context.Background(), dirUser, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify transaction committed.
	if !tx.committed {
		t.Fatal("transaction not committed")
	}

	// Verify managed user.
	if result.ManagedUser.DisplayName != "Test User" {
		t.Fatalf("ManagedUser.DisplayName = %q", result.ManagedUser.DisplayName)
	}
	if result.ManagedUser.LifecycleStatus != UserLifecycleActive {
		t.Fatalf("ManagedUser.LifecycleStatus = %q", result.ManagedUser.LifecycleStatus)
	}

	// Verify binding.
	if result.Binding.ProviderUID != "ext_001" {
		t.Fatalf("Binding.ProviderUID = %q", result.Binding.ProviderUID)
	}

	// Verify creation params.
	if gotCreateUserParams.Username != "test@example.com" {
		t.Fatalf("Username = %q", gotCreateUserParams.Username)
	}
	if gotCreateUserParams.LifecycleStatus != "active" {
		t.Fatalf("LifecycleStatus = %q", gotCreateUserParams.LifecycleStatus)
	}
	if gotCreateUserParams.UserType != "employee" {
		t.Fatalf("UserType = %q", gotCreateUserParams.UserType)
	}
	if !gotCreateUserParams.PrimarySourceID.Valid {
		t.Fatal("PrimarySourceID not valid")
	}

	// Verify binding params.
	if !gotCreateBindingParams.IsPrimary {
		t.Fatal("binding not primary")
	}
	if gotCreateBindingParams.ProviderUid != "ext_001" {
		t.Fatalf("ProviderUid = %q", gotCreateBindingParams.ProviderUid)
	}
}

func TestProvisionAndBindAutoCreateDisabled(t *testing.T) {
	tx := &mockTx{}
	q := &mockStoreQueries{}

	store := Store{
		q:          q,
		transactor: &mockTransactor{tx: tx},
	}

	dirUser := testDomainDirectoryUser()
	policy := ProvisionPolicy{
		AutoCreateManagedUsers: false,
		DefaultLifecycleStatus: UserLifecycleActive,
	}

	_, err := store.ProvisionAndBind(context.Background(), dirUser, policy)
	if err == nil {
		t.Fatal("expected error when auto-create disabled")
	}
	if !strings.Contains(err.Error(), "provisioning disabled") {
		t.Fatalf("error = %q", err.Error())
	}

	// Transaction should not have committed.
	if tx.committed {
		t.Fatal("transaction should not have committed")
	}
}

func TestProvisionAndBindCreateUserErrorRollsBack(t *testing.T) {
	tx := &mockTx{}
	txQ := &mockStoreQueries{
		createUserFn: func(_ context.Context, _ generated.CreateManagedUserParams) (generated.User, error) {
			return generated.User{}, errors.New("duplicate username")
		},
	}

	store := Store{
		q:          &mockStoreQueries{},
		transactor: &mockTransactor{tx: tx},
		txFactory:  func(pgx.Tx) Store { return Store{q: txQ} },
	}

	dirUser := testDomainDirectoryUser()
	policy := ProvisionPolicy{
		AutoCreateManagedUsers: true,
		DefaultLifecycleStatus: UserLifecycleActive,
	}

	_, err := store.ProvisionAndBind(context.Background(), dirUser, policy)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "duplicate username") {
		t.Fatalf("error = %q", err.Error())
	}

	// Verify defer Rollback was called (commit was not).
	if tx.committed {
		t.Fatal("transaction should not have committed on error")
	}
}

// --- Conversion helper tests ---

func TestToDomainDirectoryUser(t *testing.T) {
	g := testGenDirectoryUser()
	got := toDomainDirectoryUser(g)

	if got.ID != testDirID {
		t.Fatalf("ID = %q, want %q", got.ID, testDirID)
	}
	if got.EntityID != testEntityID {
		t.Fatalf("EntityID = %q, want %q", got.EntityID, testEntityID)
	}
	if got.SourceID != testSourceID {
		t.Fatalf("SourceID = %q, want %q", got.SourceID, testSourceID)
	}
	if got.ExternalUserID != "ext_001" {
		t.Fatalf("ExternalUserID = %q", got.ExternalUserID)
	}
	if got.ExternalUnionID != "union_1" {
		t.Fatalf("ExternalUnionID = %q", got.ExternalUnionID)
	}
	if got.Email != "test@example.com" {
		t.Fatalf("Email = %q", got.Email)
	}
	if got.Status != DirectoryUserStatusActive {
		t.Fatalf("Status = %q", got.Status)
	}
}

func TestToDomainManagedUser(t *testing.T) {
	g := testGenUser()
	got := toDomainManagedUser(g)

	if got.ID != testUserID {
		t.Fatalf("ID = %q, want %q", got.ID, testUserID)
	}
	if got.EntityID != testEntityID {
		t.Fatalf("EntityID = %q", got.EntityID)
	}
	if got.Username != "test@example.com" {
		t.Fatalf("Username = %q", got.Username)
	}
	if got.LifecycleStatus != UserLifecycleActive {
		t.Fatalf("LifecycleStatus = %q", got.LifecycleStatus)
	}
	if got.UserType != UserTypeEmployee {
		t.Fatalf("UserType = %q", got.UserType)
	}
}

func TestToDomainAccountBinding(t *testing.T) {
	g := testGenAccountBinding()
	got := toDomainAccountBinding(g)

	if got.ID != testBindULID {
		t.Fatalf("ID = %q", got.ID)
	}
	if got.ProviderUID != "ext_001" {
		t.Fatalf("ProviderUID = %q", got.ProviderUID)
	}
	if got.ProviderUnionID != "union_1" {
		t.Fatalf("ProviderUnionID = %q", got.ProviderUnionID)
	}
	if !got.IsPrimary {
		t.Fatal("expected IsPrimary = true")
	}
}

func TestPgTextEmpty(t *testing.T) {
	got := pgText("")
	if got.Valid {
		t.Fatal("expected Valid = false for empty string")
	}
}

func TestPgTextNonEmpty(t *testing.T) {
	got := pgText("hello")
	if !got.Valid || got.String != "hello" {
		t.Fatalf("got %v", got)
	}
}
