// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	auditmodel "github.com/smices/open-idb/internal/audit/model"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/ephemeral"
	"github.com/smices/open-idb/internal/id"
)

// --- Mock implementations ---

type mockFeishuProvider struct {
	userInfo FeishuUserInfo
	err      error
}

type fakeFeishuAuditWriter struct {
	events []auditmodel.Event
}

func (f *fakeFeishuAuditWriter) Write(_ context.Context, event auditmodel.Event) error {
	f.events = append(f.events, event)
	return nil
}

func TestFeishuOAuthStateUsesEphemeralStore(t *testing.T) {
	store := ephemeral.NewMemoryStore()
	handler := FeishuLoginHandler{}
	handler.SetEphemeralStore(store)
	state := oauthState{
		EntityID: testEntityULID,
		SourceID: testSourceULID,
		ReturnTo: "/portal",
	}

	encoded, err := handler.encodeOAuthState(context.Background(), state)
	if err != nil {
		t.Fatalf("encodeOAuthState: %v", err)
	}
	if legacyBytes, err := base64.RawURLEncoding.DecodeString(encoded); err == nil {
		var legacy oauthState
		if json.Unmarshal(legacyBytes, &legacy) == nil && legacy.EntityID != "" {
			t.Fatalf("state %q should be opaque, not legacy base64 JSON", encoded)
		}
	}

	decoded, err := handler.decodeOAuthState(context.Background(), encoded)
	if err != nil {
		t.Fatalf("decodeOAuthState: %v", err)
	}
	if decoded.EntityID != state.EntityID || decoded.SourceID != state.SourceID || decoded.ReturnTo != state.ReturnTo {
		t.Fatalf("decoded state = %#v, want %#v", decoded, state)
	}
	if _, ok, err := store.Get(context.Background(), "oidc:state:"+encoded); err != nil {
		t.Fatalf("state lookup after decode: %v", err)
	} else if ok {
		t.Fatal("state should be deleted after decode")
	}
}

func (m *mockFeishuProvider) GetUserInfoByCode(context.Context, string) (FeishuUserInfo, error) {
	return m.userInfo, m.err
}

func (m *mockFeishuProvider) GetUserInfoByAppCode(context.Context, string) (FeishuUserInfo, error) {
	return m.userInfo, m.err
}

type mockLoginQueries struct {
	upsertDirUserFn   func(context.Context, generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error)
	getBindingFn      func(context.Context, generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error)
	createUserFn      func(context.Context, generated.CreateManagedUserParams) (generated.User, error)
	createBindingFn   func(context.Context, generated.CreateAccountBindingParams) (generated.AccountBinding, error)
	updateUserFn      func(context.Context, generated.UpdateManagedUserFromDirectoryParams) (generated.User, error)
	assignRoleFn      func(context.Context, generated.AssignRoleToUserByCodeParams) error
	getSourceFn       func(context.Context, string) (generated.GetFeishuSourceByEntityRow, error)
	getEntityBySlugFn func(context.Context, string) (generated.BusinessEntity, error)
}

func (m *mockLoginQueries) UpsertDirectoryUser(ctx context.Context, arg generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
	return m.upsertDirUserFn(ctx, arg)
}
func (m *mockLoginQueries) GetAccountBindingByProviderUID(ctx context.Context, arg generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
	return m.getBindingFn(ctx, arg)
}
func (m *mockLoginQueries) CreateManagedUser(ctx context.Context, arg generated.CreateManagedUserParams) (generated.User, error) {
	return m.createUserFn(ctx, arg)
}
func (m *mockLoginQueries) CreateAccountBinding(ctx context.Context, arg generated.CreateAccountBindingParams) (generated.AccountBinding, error) {
	return m.createBindingFn(ctx, arg)
}
func (m *mockLoginQueries) UpdateManagedUserFromDirectory(ctx context.Context, arg generated.UpdateManagedUserFromDirectoryParams) (generated.User, error) {
	return m.updateUserFn(ctx, arg)
}
func (m *mockLoginQueries) AssignRoleToUserByCode(ctx context.Context, arg generated.AssignRoleToUserByCodeParams) error {
	if m.assignRoleFn != nil {
		return m.assignRoleFn(ctx, arg)
	}
	return nil
}
func (m *mockLoginQueries) GetFeishuSourceByEntity(ctx context.Context, entityID string) (generated.GetFeishuSourceByEntityRow, error) {
	return m.getSourceFn(ctx, entityID)
}
func (m *mockLoginQueries) GetEntityBySlug(ctx context.Context, slug string) (generated.BusinessEntity, error) {
	if m.getEntityBySlugFn != nil {
		return m.getEntityBySlugFn(ctx, slug)
	}
	return generated.BusinessEntity{}, nil
}

// --- Test helpers ---

var (
	testEntityULID = mustTestULID("01HZZZZZZZ0000000000000001")
	testSourceULID = mustTestULID("01HZZZZZZZ0000000000000002")
	testUserULID   = mustTestULID("01HZZZZZZZ0000000000000003")
	testDirULID    = mustTestULID("01HZZZZZZZ0000000000000004")
	testBindULID   = mustTestULID("01HZZZZZZZ0000000000000005")
)

func mustTestULID(s string) string {
	if err := id.ValidateULID(s); err != nil {
		panic(err)
	}
	return s
}

func testFeishuUserInfo() FeishuUserInfo {
	return FeishuUserInfo{
		UserID:     "emp_001",
		UnionID:    "on_union1",
		OpenID:     "ou_open1",
		Name:       "张三",
		Email:      "zhangsan@example.test",
		Phone:      "13800000000",
		AvatarURL:  "https://example.test/avatar.png",
		Status:     "active",
		RawProfile: []byte(`{"name":"张三"}`),
	}
}

func testDirectoryUser() generated.DirectoryUser {
	return generated.DirectoryUser{
		ID:              testDirULID,
		EntityID:        testEntityULID,
		SourceID:        testSourceULID,
		ExternalUserID:  "emp_001",
		ExternalUnionID: pgText("on_union1"),
		ExternalOpenID:  pgText("ou_open1"),
		Name:            "张三",
		Email:           pgText("zhangsan@example.test"),
		Phone:           pgText("13800000000"),
		AvatarUrl:       pgText("https://example.test/avatar.png"),
		Status:          "active",
		RawProfile:      []byte(`{"name":"张三"}`),
	}
}

func testManagedUser(lifecycleStatus string) generated.User {
	return generated.User{
		ID:              testUserULID,
		EntityID:        testEntityULID,
		Username:        "zhangsan@example.test",
		DisplayName:     "张三",
		Email:           pgText("zhangsan@example.test"),
		Phone:           pgText("13800000000"),
		AvatarUrl:       pgText("https://example.test/avatar.png"),
		LifecycleStatus: lifecycleStatus,
		UserType:        "employee",
		PrimarySourceID: pgText(testSourceULID),
		Locale:          pgText("en-US"),
	}
}

func testBinding() generated.AccountBinding {
	return generated.AccountBinding{
		ID:              testBindULID,
		EntityID:        testEntityULID,
		UserID:          testUserULID,
		SourceID:        testSourceULID,
		DirectoryUserID: testDirULID,
		ProviderUid:     "emp_001",
		ProviderUnionID: pgText("on_union1"),
		IsPrimary:       true,
	}
}

// --- Tests ---

func TestFeishuLoginExistingUserUpdatesAndCreatesSession(t *testing.T) {
	var gotUpdateParams generated.UpdateManagedUserFromDirectoryParams

	queries := &mockLoginQueries{
		upsertDirUserFn: func(_ context.Context, arg generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			if arg.ExternalUserID != "emp_001" {
				t.Fatalf("ExternalUserID = %q", arg.ExternalUserID)
			}
			return testDirectoryUser(), nil
		},
		getBindingFn: func(_ context.Context, arg generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			if arg.ProviderUid != "emp_001" {
				t.Fatalf("ProviderUid = %q", arg.ProviderUid)
			}
			return testBinding(), nil
		},
		updateUserFn: func(_ context.Context, arg generated.UpdateManagedUserFromDirectoryParams) (generated.User, error) {
			gotUpdateParams = arg
			return testManagedUser("active"), nil
		},
	}
	provider := &mockFeishuProvider{userInfo: testFeishuUserInfo()}

	svc := &FeishuLoginService{
		queries:      queries,
		feishuClient: provider,
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}

	result, err := svc.LoginViaOAuth(context.Background(),
		"01HZZZZZZZ0000000000000001",
		"01HZZZZZZZ0000000000000002",
		"auth-code",
	)
	if err != nil {
		t.Fatalf("LoginViaOAuth error = %v", err)
	}

	if gotUpdateParams.DisplayName != "张三" {
		t.Fatalf("update DisplayName = %q", gotUpdateParams.DisplayName)
	}
	if gotUpdateParams.Username.Valid {
		t.Fatalf("update Username should be null for existing bindings, got %q", gotUpdateParams.Username.String)
	}
	if result.UserID != "01HZZZZZZZ0000000000000003" {
		t.Fatalf("UserID = %q", result.UserID)
	}
	if result.EntityID != "01HZZZZZZZ0000000000000001" {
		t.Fatalf("EntityID = %q", result.EntityID)
	}
	if result.Username != "zhangsan@example.test" {
		t.Fatalf("Username = %q", result.Username)
	}
	if result.DisplayName != "张三" {
		t.Fatalf("DisplayName = %q", result.DisplayName)
	}
	if result.SessionValue == "" {
		t.Fatal("SessionValue is empty")
	}
	if result.ExpiresIn != 24*time.Hour {
		t.Fatalf("ExpiresIn = %v", result.ExpiresIn)
	}

	// Verify session can be decoded.
	session, err := DecodeSession(result.SessionValue)
	if err != nil {
		t.Fatalf("DecodeSession error = %v", err)
	}
	if session.UserID != "01HZZZZZZZ0000000000000003" {
		t.Fatalf("session UserID = %q", session.UserID)
	}
	if session.Username != "zhangsan@example.test" {
		t.Fatalf("session Username = %q", session.Username)
	}
}

func TestFeishuLoginNewUserCreatesUserAndBinding(t *testing.T) {
	var gotCreateUserParams generated.CreateManagedUserParams
	var gotCreateBindingParams generated.CreateAccountBindingParams

	queries := &mockLoginQueries{
		upsertDirUserFn: func(_ context.Context, _ generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			return testDirectoryUser(), nil
		},
		getBindingFn: func(_ context.Context, _ generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			return generated.AccountBinding{}, pgx.ErrNoRows
		},
		createUserFn: func(_ context.Context, arg generated.CreateManagedUserParams) (generated.User, error) {
			gotCreateUserParams = arg
			return testManagedUser("active"), nil
		},
		createBindingFn: func(_ context.Context, arg generated.CreateAccountBindingParams) (generated.AccountBinding, error) {
			gotCreateBindingParams = arg
			return testBinding(), nil
		},
	}
	provider := &mockFeishuProvider{userInfo: testFeishuUserInfo()}

	svc := &FeishuLoginService{
		queries:      queries,
		feishuClient: provider,
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}

	result, err := svc.LoginViaOAuth(context.Background(),
		"01HZZZZZZZ0000000000000001",
		"01HZZZZZZZ0000000000000002",
		"auth-code",
	)
	if err != nil {
		t.Fatalf("LoginViaOAuth error = %v", err)
	}

	// Verify managed user creation params.
	if gotCreateUserParams.Username != "zhangsan@example.test" {
		t.Fatalf("Username = %q", gotCreateUserParams.Username)
	}
	if gotCreateUserParams.DisplayName != "张三" {
		t.Fatalf("DisplayName = %q", gotCreateUserParams.DisplayName)
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

	// Verify binding creation params.
	if gotCreateBindingParams.ProviderUid != "emp_001" {
		t.Fatalf("ProviderUid = %q", gotCreateBindingParams.ProviderUid)
	}
	if !gotCreateBindingParams.IsPrimary {
		t.Fatal("binding not primary")
	}

	if result.SessionValue == "" {
		t.Fatal("SessionValue is empty")
	}
}

func TestFeishuLoginNewUserFallsBackToUserIDAsUsername(t *testing.T) {
	var gotCreateUserParams generated.CreateManagedUserParams

	info := testFeishuUserInfo()
	info.Email = "" // No email

	queries := &mockLoginQueries{
		upsertDirUserFn: func(_ context.Context, _ generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			return testDirectoryUser(), nil
		},
		getBindingFn: func(_ context.Context, _ generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			return generated.AccountBinding{}, pgx.ErrNoRows
		},
		createUserFn: func(_ context.Context, arg generated.CreateManagedUserParams) (generated.User, error) {
			gotCreateUserParams = arg
			user := testManagedUser("active")
			user.Username = arg.Username
			return user, nil
		},
		createBindingFn: func(_ context.Context, _ generated.CreateAccountBindingParams) (generated.AccountBinding, error) {
			return testBinding(), nil
		},
	}
	provider := &mockFeishuProvider{userInfo: info}

	svc := &FeishuLoginService{
		queries:      queries,
		feishuClient: provider,
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}

	_, err := svc.LoginViaOAuth(context.Background(),
		"01HZZZZZZZ0000000000000001",
		"01HZZZZZZZ0000000000000002",
		"auth-code",
	)
	if err != nil {
		t.Fatalf("LoginViaOAuth error = %v", err)
	}

	if gotCreateUserParams.Username != "emp_001" {
		t.Fatalf("Username = %q, want emp_001 (fallback to UserID)", gotCreateUserParams.Username)
	}
}

func TestFeishuLoginDisabledUserRejected(t *testing.T) {
	queries := &mockLoginQueries{
		upsertDirUserFn: func(_ context.Context, _ generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			return testDirectoryUser(), nil
		},
		getBindingFn: func(_ context.Context, _ generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			return testBinding(), nil
		},
		updateUserFn: func(_ context.Context, _ generated.UpdateManagedUserFromDirectoryParams) (generated.User, error) {
			return testManagedUser("disabled"), nil
		},
	}
	provider := &mockFeishuProvider{userInfo: testFeishuUserInfo()}

	svc := &FeishuLoginService{
		queries:      queries,
		feishuClient: provider,
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}

	_, err := svc.LoginViaOAuth(context.Background(),
		"01HZZZZZZZ0000000000000001",
		"01HZZZZZZZ0000000000000002",
		"auth-code",
	)
	if err == nil {
		t.Fatal("expected error for disabled user")
	}
	if err.Error() != "user_disabled" {
		t.Fatalf("error = %q, want user_disabled", err.Error())
	}
}

func TestFeishuLoginLockedUserRejected(t *testing.T) {
	queries := &mockLoginQueries{
		upsertDirUserFn: func(_ context.Context, _ generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			return testDirectoryUser(), nil
		},
		getBindingFn: func(_ context.Context, _ generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			return testBinding(), nil
		},
		updateUserFn: func(_ context.Context, _ generated.UpdateManagedUserFromDirectoryParams) (generated.User, error) {
			return testManagedUser("locked"), nil
		},
	}
	provider := &mockFeishuProvider{userInfo: testFeishuUserInfo()}

	svc := &FeishuLoginService{
		queries:      queries,
		feishuClient: provider,
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}

	_, err := svc.LoginViaOAuth(context.Background(),
		"01HZZZZZZZ0000000000000001",
		"01HZZZZZZZ0000000000000002",
		"auth-code",
	)
	if err == nil {
		t.Fatal("expected error for locked user")
	}
	if err.Error() != "user_disabled" {
		t.Fatalf("error = %q, want user_disabled", err.Error())
	}
}

func TestFeishuLoginViaAppCode(t *testing.T) {
	queries := &mockLoginQueries{
		upsertDirUserFn: func(_ context.Context, _ generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			return testDirectoryUser(), nil
		},
		getBindingFn: func(_ context.Context, _ generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			return generated.AccountBinding{}, pgx.ErrNoRows
		},
		createUserFn: func(_ context.Context, _ generated.CreateManagedUserParams) (generated.User, error) {
			return testManagedUser("active"), nil
		},
		createBindingFn: func(_ context.Context, _ generated.CreateAccountBindingParams) (generated.AccountBinding, error) {
			return testBinding(), nil
		},
	}
	provider := &mockFeishuProvider{userInfo: testFeishuUserInfo()}

	svc := &FeishuLoginService{
		queries:      queries,
		feishuClient: provider,
		sessionTTL:   12 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}

	result, err := svc.LoginViaAppCode(context.Background(),
		"01HZZZZZZZ0000000000000001",
		"01HZZZZZZZ0000000000000002",
		"app-auth-code",
		"dashboard",
	)
	if err != nil {
		t.Fatalf("LoginViaAppCode error = %v", err)
	}
	if result.SessionValue == "" {
		t.Fatal("SessionValue is empty")
	}
	if result.ExpiresIn != 12*time.Hour {
		t.Fatalf("ExpiresIn = %v, want 12h", result.ExpiresIn)
	}
}

func TestFeishuLoginProviderErrorPropagates(t *testing.T) {
	queries := &mockLoginQueries{}
	provider := &mockFeishuProvider{err: errors.New("feishu api down")}

	svc := &FeishuLoginService{
		queries:      queries,
		feishuClient: provider,
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}

	_, err := svc.LoginViaOAuth(context.Background(),
		"01HZZZZZZZ0000000000000001",
		"01HZZZZZZZ0000000000000002",
		"auth-code",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "feishu api down") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestFeishuLoginLookupSourceID(t *testing.T) {
	queries := &mockLoginQueries{
		getSourceFn: func(_ context.Context, entityID string) (generated.GetFeishuSourceByEntityRow, error) {
			return generated.GetFeishuSourceByEntityRow{
				ID:          testSourceULID,
				EntityID:    entityID,
				Type:        "feishu",
				Name:        "Feishu",
				Status:      "active",
				SyncEnabled: true,
			}, nil
		},
	}

	svc := &FeishuLoginService{
		queries:      queries,
		feishuClient: &mockFeishuProvider{},
		sessionTTL:   24 * time.Hour,
	}

	sourceID, err := svc.LookupSourceID(context.Background(), "01HZZZZZZZ0000000000000001")
	if err != nil {
		t.Fatalf("LookupSourceID error = %v", err)
	}
	if sourceID != "01HZZZZZZZ0000000000000002" {
		t.Fatalf("sourceID = %q", sourceID)
	}
}

func TestFeishuLoginInvalidEntityID(t *testing.T) {
	svc := &FeishuLoginService{
		queries:      &mockLoginQueries{},
		feishuClient: &mockFeishuProvider{},
		sessionTTL:   24 * time.Hour,
	}

	_, err := svc.LoginViaOAuth(context.Background(), "not-a-ulid", "01HZZZZZZZ0000000000000002", "code")
	if err == nil {
		t.Fatal("expected error for invalid entity ID")
	}
}

func TestClassifyFeishuLoginError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "disabled user", err: errors.New("user_disabled"), want: "user_disabled"},
		{name: "no account", err: errors.New("no_account"), want: "no_account"},
		{name: "config", err: errors.New("feishu client is not configured"), want: "feishu_config_error"},
		{name: "app token", err: errors.New("feishu oauth: feishu app token failed: code=999 msg=bad secret"), want: "feishu_app_token_failed"},
		{name: "oidc token", err: errors.New("feishu oauth: feishu oidc token exchange failed: code=20027 msg=invalid code"), want: "feishu_oidc_token_failed"},
		{name: "app code", err: errors.New("feishu app code: feishu app token exchange failed: code=400 msg=invalid auth code"), want: "feishu_app_code_failed"},
		{name: "user info", err: errors.New("feishu oauth: feishu user info failed: code=99991663 msg=permission denied"), want: "feishu_user_info_failed"},
		{name: "directory user upsert", err: errors.New("upsert directory user: duplicate key value violates unique constraint"), want: "feishu_directory_user_upsert_failed"},
		{name: "binding lookup", err: errors.New("lookup binding: database unavailable"), want: "feishu_binding_lookup_failed"},
		{name: "managed user update", err: errors.New("update managed user: database unavailable"), want: "feishu_managed_user_update_failed"},
		{name: "managed user create", err: errors.New("create managed user: duplicate key value violates unique constraint"), want: "feishu_managed_user_create_failed"},
		{name: "role assignment", err: errors.New("assign default employee role: role not found"), want: "feishu_role_assign_failed"},
		{name: "binding create", err: errors.New("create binding: duplicate key value violates unique constraint"), want: "feishu_binding_create_failed"},
		{name: "account fallback", err: errors.New("account state is invalid"), want: "feishu_account_error"},
		{name: "session", err: errors.New("create session: database unavailable"), want: "feishu_session_failed"},
		{name: "fallback", err: errors.New("unexpected provider outage"), want: "feishu_login_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyFeishuLoginError(tt.err); got != tt.want {
				t.Fatalf("classifyFeishuLoginError() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Handler tests ---

func TestFeishuCallbackSetsSessionAndRedirects(t *testing.T) {
	queries := &mockLoginQueries{
		upsertDirUserFn: func(_ context.Context, _ generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			return testDirectoryUser(), nil
		},
		getBindingFn: func(_ context.Context, _ generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			return testBinding(), nil
		},
		updateUserFn: func(_ context.Context, _ generated.UpdateManagedUserFromDirectoryParams) (generated.User, error) {
			return testManagedUser("active"), nil
		},
		getSourceFn: func(_ context.Context, _ string) (generated.GetFeishuSourceByEntityRow, error) {
			return generated.GetFeishuSourceByEntityRow{
				ID:       testSourceULID,
				EntityID: testEntityULID,
				Type:     "feishu",
				Name:     "Feishu",
				Status:   "active",
			}, nil
		},
	}
	provider := &mockFeishuProvider{userInfo: testFeishuUserInfo()}

	loginSvc := &FeishuLoginService{
		queries:      queries,
		feishuClient: provider,
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}
	providerSvc := &LoginProviderService{
		queries:           &mockProviderQueries{},
		feishuAppID:       "test-app",
		feishuRedirectURI: "https://example.test/callback",
	}
	handler := NewFeishuLoginHandler(loginSvc, providerSvc, "test-app", "https://example.test/callback")

	// Build state parameter.
	state := oauthState{
		EntityID: "01HZZZZZZZ0000000000000001",
		SourceID: "01HZZZZZZZ0000000000000002",
		ReturnTo: "/portal",
	}
	stateBytes, _ := json.Marshal(state)
	stateEncoded := base64.RawURLEncoding.EncodeToString(stateBytes)

	router := chiRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet,
		"/auth/feishu/callback?code=test-code&state="+stateEncoded, nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if location := rec.Header().Get("Location"); location != "/portal" {
		t.Fatalf("Location = %q, want /portal", location)
	}

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "idb_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("missing idb_session cookie")
	}

	// Verify the session cookie can be decoded.
	session, err := DecodeSession(sessionCookie.Value)
	if err != nil {
		t.Fatalf("DecodeSession error = %v", err)
	}
	if session.Username != "zhangsan@example.test" {
		t.Fatalf("session Username = %q", session.Username)
	}
}

func TestFeishuCallbackRedirectsBrowserToConfiguredWebBaseURL(t *testing.T) {
	queries := &mockLoginQueries{
		upsertDirUserFn: func(_ context.Context, _ generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			return testDirectoryUser(), nil
		},
		getBindingFn: func(_ context.Context, _ generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			return testBinding(), nil
		},
		updateUserFn: func(_ context.Context, _ generated.UpdateManagedUserFromDirectoryParams) (generated.User, error) {
			return testManagedUser("active"), nil
		},
		getSourceFn: func(_ context.Context, _ string) (generated.GetFeishuSourceByEntityRow, error) {
			return generated.GetFeishuSourceByEntityRow{
				ID:       testSourceULID,
				EntityID: testEntityULID,
				Type:     "feishu",
				Name:     "Feishu",
				Status:   "active",
			}, nil
		},
	}
	loginSvc := &FeishuLoginService{
		queries:      queries,
		feishuClient: &mockFeishuProvider{userInfo: testFeishuUserInfo()},
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}
	providerSvc := &LoginProviderService{
		queries:     &mockProviderQueries{},
		feishuAppID: "test-app",
	}
	handler := NewFeishuLoginHandler(loginSvc, providerSvc, "test-app", "https://example.test/callback")
	handler.SetWebBaseURL("http://localhost:5180")

	state := oauthState{
		EntityID: "01HZZZZZZZ0000000000000001",
		SourceID: "01HZZZZZZZ0000000000000002",
		ReturnTo: "/portal",
	}
	stateBytes, _ := json.Marshal(state)
	stateEncoded := base64.RawURLEncoding.EncodeToString(stateBytes)

	router := chiRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/auth/feishu/callback?code=test-code&state="+stateEncoded, nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if location := rec.Header().Get("Location"); location != "http://localhost:5180/portal" {
		t.Fatalf("Location = %q, want http://localhost:5180/portal", location)
	}
}

func TestFeishuCallbackFailureRedirectIncludesReasonAndTraceID(t *testing.T) {
	audit := &fakeFeishuAuditWriter{}
	loginSvc := &FeishuLoginService{
		queries:      &mockLoginQueries{},
		feishuClient: &mockFeishuProvider{err: errors.New("feishu user info failed: code=99991663 msg=permission denied")},
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}
	providerSvc := &LoginProviderService{
		queries:     &mockProviderQueries{},
		feishuAppID: "test-app",
	}
	handler := NewFeishuLoginHandler(loginSvc, providerSvc, "test-app", "https://example.test/callback", audit)

	state := oauthState{
		EntityID: "01HZZZZZZZ0000000000000001",
		SourceID: "01HZZZZZZZ0000000000000002",
		ReturnTo: "/portal",
	}
	stateBytes, _ := json.Marshal(state)
	stateEncoded := base64.RawURLEncoding.EncodeToString(stateBytes)

	router := chiRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/auth/feishu/callback?code=test-code&state="+stateEncoded, nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect Location %q: %v", location, err)
	}
	if got := parsed.Query().Get("login_error"); got != "feishu_user_info_failed" {
		t.Fatalf("login_error = %q, want feishu_user_info_failed", got)
	}
	traceID := parsed.Query().Get("trace_id")
	if err := id.ValidateULID(traceID); err != nil {
		t.Fatalf("trace_id = %q is not a ULID: %v", traceID, err)
	}
	if len(audit.events) != 1 {
		t.Fatalf("audit event count = %d, want 1", len(audit.events))
	}
	after, ok := audit.events[0].After.(map[string]string)
	if !ok {
		t.Fatalf("audit After type = %T, want map[string]string", audit.events[0].After)
	}
	if after["reason_code"] != "feishu_user_info_failed" {
		t.Fatalf("audit reason_code = %q, want feishu_user_info_failed", after["reason_code"])
	}
	if after["trace_id"] != traceID || audit.events[0].TraceID != traceID {
		t.Fatalf("audit trace mismatch: after=%q event=%q redirect=%q", after["trace_id"], audit.events[0].TraceID, traceID)
	}
	if !strings.Contains(after["reason"], "permission denied") {
		t.Fatalf("audit reason = %q, want provider details", after["reason"])
	}
}

func TestFeishuCallbackResolvesSourceIDWhenMissing(t *testing.T) {
	queries := &mockLoginQueries{
		upsertDirUserFn: func(_ context.Context, _ generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			return testDirectoryUser(), nil
		},
		getBindingFn: func(_ context.Context, _ generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			return testBinding(), nil
		},
		updateUserFn: func(_ context.Context, _ generated.UpdateManagedUserFromDirectoryParams) (generated.User, error) {
			return testManagedUser("active"), nil
		},
		getSourceFn: func(_ context.Context, _ string) (generated.GetFeishuSourceByEntityRow, error) {
			return generated.GetFeishuSourceByEntityRow{
				ID:       testSourceULID,
				EntityID: testEntityULID,
				Type:     "feishu",
				Name:     "Feishu",
				Status:   "active",
			}, nil
		},
	}
	provider := &mockFeishuProvider{userInfo: testFeishuUserInfo()}

	loginSvc := &FeishuLoginService{
		queries:      queries,
		feishuClient: provider,
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}
	providerSvc := &LoginProviderService{
		queries:     &mockProviderQueries{},
		feishuAppID: "test-app",
	}
	handler := NewFeishuLoginHandler(loginSvc, providerSvc, "test-app", "https://example.test/callback")

	// State with only entity_id (no source_id).
	state := oauthState{
		EntityID: "01HZZZZZZZ0000000000000001",
	}
	stateBytes, _ := json.Marshal(state)
	stateEncoded := base64.RawURLEncoding.EncodeToString(stateBytes)

	router := chiRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet,
		"/auth/feishu/callback?code=test-code&state="+stateEncoded, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestFeishuExchangeEndpoint(t *testing.T) {
	queries := &mockLoginQueries{
		upsertDirUserFn: func(_ context.Context, _ generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error) {
			return testDirectoryUser(), nil
		},
		getBindingFn: func(_ context.Context, _ generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error) {
			return generated.AccountBinding{}, pgx.ErrNoRows
		},
		createUserFn: func(_ context.Context, _ generated.CreateManagedUserParams) (generated.User, error) {
			return testManagedUser("active"), nil
		},
		createBindingFn: func(_ context.Context, _ generated.CreateAccountBindingParams) (generated.AccountBinding, error) {
			return testBinding(), nil
		},
		getSourceFn: func(_ context.Context, _ string) (generated.GetFeishuSourceByEntityRow, error) {
			return generated.GetFeishuSourceByEntityRow{
				ID:       testSourceULID,
				EntityID: testEntityULID,
				Type:     "feishu",
				Name:     "Feishu",
				Status:   "active",
			}, nil
		},
	}
	provider := &mockFeishuProvider{userInfo: testFeishuUserInfo()}

	loginSvc := &FeishuLoginService{
		queries:      queries,
		feishuClient: provider,
		sessionTTL:   24 * time.Hour,
		policy:       LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}
	providerSvc := &LoginProviderService{
		queries:     &mockProviderQueries{},
		feishuAppID: "test-app",
	}
	handler := NewFeishuLoginHandler(loginSvc, providerSvc, "test-app", "https://example.test/callback")

	router := chiRouter()
	handler.RegisterRoutes(router)

	body := `{"auth_code":"app-code-123","entity_id":"01HZZZZZZZ0000000000000001"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/feishu/exchange", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["session"] == "" {
		t.Fatal("response missing session")
	}
	if resp["username"] != "zhangsan@example.test" {
		t.Fatalf("username = %q", resp["username"])
	}
}

// chiRouter creates a chi.Router for handler tests.
func chiRouter() *chi.Mux {
	return chi.NewRouter()
}
