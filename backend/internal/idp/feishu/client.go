// SPDX-License-Identifier: MIT

package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
	"github.com/smices/open-idb/internal/idp"
)

const defaultBaseURL = "https://open.feishu.cn"
const defaultPageSize = 50

type Config struct {
	AppID     string
	AppSecret string
	BaseURL   string
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

type feishuPaginatedResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items         []json.RawMessage `json:"items"`
		HasMore       bool              `json:"has_more"`
		PageToken     string            `json:"page_token"`
		NextPageToken string            `json:"next_page_token"`
	} `json:"data"`
}

func NewClient(cfg Config, httpClient *http.Client) (*Client, error) {
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, fmt.Errorf("feishu app id and app secret are required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
		return nil, fmt.Errorf("feishu base url is invalid: %w", err)
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{cfg: cfg, httpClient: httpClient}, nil
}

func (c *Client) FullSync(ctx context.Context) (idp.FullSyncData, error) {
	token, err := c.entityAccessToken(ctx)
	if err != nil {
		return idp.FullSyncData{}, err
	}
	departments, err := c.departments(ctx, token)
	if err != nil {
		return idp.FullSyncData{}, err
	}
	users, err := c.users(ctx, token, departments)
	if err != nil {
		return idp.FullSyncData{}, err
	}
	return idp.FullSyncData{Departments: departments, Users: users}, nil
}

// IncrementalSync consumes a batch of webhook events and fetches changed users/departments.
// Deletion events are represented as status-deleted records for upsert.
func (c *Client) IncrementalSync(ctx context.Context, events []idp.DirectorySyncEvent) (idp.FullSyncData, error) {
	if len(events) == 0 {
		return idp.FullSyncData{}, nil
	}

	token, err := c.entityAccessToken(ctx)
	if err != nil {
		return idp.FullSyncData{}, err
	}

	departments := make([]idp.DirectoryDepartment, 0)
	users := make([]idp.DirectoryUser, 0)
	departmentIDs := make(map[string]struct{})
	userIDs := make(map[string]struct{})

	for _, event := range events {
		eventType := strings.ToLower(strings.TrimSpace(event.EventType))
		if eventType == "" || !event.IsKnown() {
			continue
		}

		objectType := strings.ToLower(strings.TrimSpace(event.ObjectType))
		if objectType == "user" {
			if event.ObjectID == "" {
				continue
			}
			user, err := c.getUserByID(ctx, token, event.ObjectID, event.ObjectIDType)
			if err != nil {
				if event.IsDeleteEvent() {
					if _, ok := userIDs[event.ObjectID]; !ok {
						users = append(users, idp.DirectoryUser{
							ExternalUserID: event.ObjectID,
							Status:         "deleted",
							RawProfile:     cloneBytes([]byte(`{}`)),
						})
						userIDs[event.ObjectID] = struct{}{}
					}
					continue
				}
				return idp.FullSyncData{}, err
			}
			if _, ok := userIDs[user.ExternalUserID]; !ok {
				users = append(users, user)
				userIDs[user.ExternalUserID] = struct{}{}
			}
			continue
		}

		if objectType == "department" {
			if event.ObjectID == "" {
				continue
			}
			department, err := c.getDepartmentByID(ctx, token, event.ObjectID, event.ObjectIDType)
			if err != nil {
				if event.IsDeleteEvent() {
					if _, ok := departmentIDs[event.ObjectID]; !ok {
						departments = append(departments, idp.DirectoryDepartment{
							ExternalDepartmentID:       event.ObjectID,
							ParentExternalDepartmentID: "",
							Name:                       "",
							RawProfile:                 cloneBytes([]byte(`{}`)),
						})
						departmentIDs[event.ObjectID] = struct{}{}
					}
					continue
				}
				return idp.FullSyncData{}, err
			}
			if _, ok := departmentIDs[department.ExternalDepartmentID]; !ok {
				departments = append(departments, department)
				departmentIDs[department.ExternalDepartmentID] = struct{}{}
			}
			continue
		}
	}

	return idp.FullSyncData{Departments: departments, Users: users}, nil
}

func (c *Client) entityAccessToken(ctx context.Context) (string, error) {
	payload := map[string]string{
		"app_id":     c.cfg.AppID,
		"app_secret": c.cfg.AppSecret,
	}
	var response struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/open-apis/auth/v3/tenant_access_token/internal", "", payload, &response); err != nil {
		return "", err
	}
	if response.Code != 0 {
		return "", fmt.Errorf("feishu entity token failed: code=%d msg=%s", response.Code, response.Msg)
	}
	if response.TenantAccessToken == "" {
		return "", fmt.Errorf("feishu entity token response missing token")
	}
	return response.TenantAccessToken, nil
}

func (c *Client) departments(ctx context.Context, token string) ([]idp.DirectoryDepartment, error) {
	type rawDepartment struct {
		ExternalID string
		ParentID   string
		Name       string
		Raw        []byte
	}
	rawDepartments := make([]rawDepartment, 0)
	openIDToDepartmentID := make(map[string]string)
	pageToken := ""

	for {
		response, err := c.departmentsPage(ctx, token, pageToken)
		if err != nil {
			return nil, err
		}

		for _, raw := range response.Data.Items {
			var item struct {
				DepartmentID     string `json:"department_id"`
				OpenDepartmentID string `json:"open_department_id"`
				ParentID         string `json:"parent_department_id"`
				Name             string `json:"name"`
			}
			if err := json.Unmarshal(raw, &item); err != nil {
				return nil, err
			}
			externalID := firstNonEmpty(item.DepartmentID, item.OpenDepartmentID)
			if item.OpenDepartmentID != "" && externalID != "" {
				openIDToDepartmentID[item.OpenDepartmentID] = externalID
			}
			rawDepartments = append(rawDepartments, rawDepartment{
				ExternalID: externalID,
				ParentID:   item.ParentID,
				Name:       item.Name,
				Raw:        cloneBytes(raw),
			})
		}

		if !response.Data.HasMore {
			break
		}
		nextPageToken := firstNonEmpty(response.Data.NextPageToken, response.Data.PageToken)
		if nextPageToken == "" {
			return nil, fmt.Errorf("feishu departments response missing page_token while has_more=true")
		}
		pageToken = nextPageToken
	}

	out := make([]idp.DirectoryDepartment, 0, len(rawDepartments))
	for _, item := range rawDepartments {
		parentID := item.ParentID
		if mapped, ok := openIDToDepartmentID[parentID]; ok {
			parentID = mapped
		}
		out = append(out, idp.DirectoryDepartment{
			ExternalDepartmentID:       item.ExternalID,
			ParentExternalDepartmentID: parentID,
			Name:                       item.Name,
			RawProfile:                 cloneBytes(item.Raw),
		})
	}
	return out, nil
}

func (c *Client) departmentsPage(ctx context.Context, token string, pageToken string) (feishuPaginatedResponse, error) {
	var response feishuPaginatedResponse
	query := url.Values{}
	query.Set("fetch_child", "true")
	query.Set("department_id_type", "department_id")
	query.Set("page_size", strconv.Itoa(defaultPageSize))
	if pageToken != "" {
		query.Set("page_token", pageToken)
	}
	if err := c.doJSON(ctx, http.MethodGet, "/open-apis/contact/v3/departments/0/children?"+query.Encode(), token, nil, &response); err != nil {
		return feishuPaginatedResponse{}, err
	}
	if response.Code != 0 {
		return feishuPaginatedResponse{}, fmt.Errorf("feishu departments failed: code=%d msg=%s", response.Code, response.Msg)
	}
	return response, nil
}

func (c *Client) users(ctx context.Context, token string, departments []idp.DirectoryDepartment) ([]idp.DirectoryUser, error) {
	departmentIDs := []string{"0"}
	seenDepartments := map[string]struct{}{"0": {}}
	for _, department := range departments {
		departmentID := strings.TrimSpace(department.ExternalDepartmentID)
		if departmentID == "" {
			continue
		}
		if _, ok := seenDepartments[departmentID]; ok {
			continue
		}
		departmentIDs = append(departmentIDs, departmentID)
		seenDepartments[departmentID] = struct{}{}
	}

	out := make([]idp.DirectoryUser, 0)
	seenUsers := make(map[string]struct{})
	for _, departmentID := range departmentIDs {
		users, err := c.usersByDepartment(ctx, token, departmentID)
		if err != nil {
			return nil, err
		}
		for _, user := range users {
			userKey := firstNonEmpty(user.ExternalUserID, user.ExternalOpenID, user.ExternalUnionID)
			if userKey == "" {
				continue
			}
			if _, ok := seenUsers[userKey]; ok {
				continue
			}
			out = append(out, user)
			seenUsers[userKey] = struct{}{}
		}
	}

	return out, nil
}

func (c *Client) usersByDepartment(ctx context.Context, token string, departmentID string) ([]idp.DirectoryUser, error) {
	out := make([]idp.DirectoryUser, 0)
	pageToken := ""

	for {
		response, err := c.usersByDepartmentPage(ctx, token, departmentID, pageToken)
		if err != nil {
			return nil, err
		}

		for _, raw := range response.Data.Items {
			var item struct {
				UserID     string          `json:"user_id"`
				UnionID    string          `json:"union_id"`
				OpenID     string          `json:"open_id"`
				Name       string          `json:"name"`
				EnName     string          `json:"en_name"`
				English    string          `json:"english_name"`
				NameEN     string          `json:"name_en"`
				I18nName   json.RawMessage `json:"i18n_name"`
				Email      string          `json:"email"`
				Mobile     string          `json:"mobile"`
				AvatarURL  string          `json:"avatar_url"`
				EmployeeNo string          `json:"employee_no"`
				JobTitle   string          `json:"job_title"`
				Status     struct {
					IsActivated bool `json:"is_activated"`
					IsFrozen    bool `json:"is_frozen"`
					IsResigned  bool `json:"is_resigned"`
				} `json:"status"`
			}
			if err := json.Unmarshal(raw, &item); err != nil {
				return nil, err
			}
			out = append(out, idp.DirectoryUser{
				ExternalUserID:  firstNonEmpty(item.UserID, item.OpenID, item.UnionID),
				ExternalUnionID: item.UnionID,
				ExternalOpenID:  item.OpenID,
				Name:            item.Name,
				EnglishName:     englishNameForUser(item.Name, item.EnName, item.English, item.NameEN, item.I18nName),
				EmployeeNo:      item.EmployeeNo,
				JobTitle:        item.JobTitle,
				Email:           item.Email,
				Phone:           item.Mobile,
				AvatarURL:       item.AvatarURL,
				Status:          mapUserStatus(item.Status.IsActivated, item.Status.IsFrozen, item.Status.IsResigned),
				RawProfile:      rawProfileWithDepartment(raw, departmentID),
			})
		}

		if !response.Data.HasMore {
			break
		}
		nextPageToken := firstNonEmpty(response.Data.NextPageToken, response.Data.PageToken)
		if nextPageToken == "" {
			return nil, fmt.Errorf("feishu users response missing page_token while has_more=true")
		}
		pageToken = nextPageToken
	}

	return out, nil
}

func (c *Client) usersByDepartmentPage(ctx context.Context, token string, departmentID string, pageToken string) (feishuPaginatedResponse, error) {
	var response feishuPaginatedResponse
	query := url.Values{}
	query.Set("department_id", departmentID)
	query.Set("department_id_type", departmentIDType(departmentID))
	query.Set("user_id_type", "user_id")
	query.Set("page_size", strconv.Itoa(defaultPageSize))
	if pageToken != "" {
		query.Set("page_token", pageToken)
	}
	if err := c.doJSON(ctx, http.MethodGet, "/open-apis/contact/v3/users/find_by_department?"+query.Encode(), token, nil, &response); err != nil {
		return feishuPaginatedResponse{}, err
	}
	if response.Code != 0 {
		return feishuPaginatedResponse{}, fmt.Errorf("feishu users failed: code=%d msg=%s", response.Code, response.Msg)
	}
	return response, nil
}

func rawProfileWithDepartment(raw []byte, departmentID string) []byte {
	departmentID = strings.TrimSpace(departmentID)
	if departmentID == "" {
		return cloneBytes(raw)
	}
	var profile map[string]interface{}
	if err := json.Unmarshal(raw, &profile); err != nil {
		return cloneBytes(raw)
	}
	profile["department_ids"] = appendStringValue(profile["department_ids"], departmentID)
	profile["department_id"] = departmentID
	normalized, err := json.Marshal(profile)
	if err != nil {
		return cloneBytes(raw)
	}
	return normalized
}

func appendStringValue(value interface{}, next string) []string {
	values := make([]string, 0, 1)
	appendValue := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		for _, existing := range values {
			if existing == candidate {
				return
			}
		}
		values = append(values, candidate)
	}
	switch typed := value.(type) {
	case []interface{}:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				appendValue(text)
			}
		}
	case []string:
		for _, item := range typed {
			appendValue(item)
		}
	case string:
		appendValue(typed)
	}
	appendValue(next)
	return values
}

func departmentIDType(departmentID string) string {
	if departmentID == "0" || strings.HasPrefix(departmentID, "od-") || strings.HasPrefix(departmentID, "od_") {
		return "open_department_id"
	}
	return "department_id"
}

func (c *Client) getUserByID(ctx context.Context, token string, userID string, idType string) (idp.DirectoryUser, error) {
	data, err := c.getObjectByID(ctx, token, "users", userID, "user_id_type", idType)
	if err != nil {
		return idp.DirectoryUser{}, err
	}
	var envelope struct {
		User json.RawMessage `json:"user"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return idp.DirectoryUser{}, err
	}
	rawUser := firstNonEmptyRaw(envelope.User)
	if len(rawUser) == 0 {
		rawUser = cloneBytes(data)
	}
	var item struct {
		UserID    string          `json:"user_id"`
		UnionID   string          `json:"union_id"`
		OpenID    string          `json:"open_id"`
		Name      string          `json:"name"`
		Email     string          `json:"email"`
		Mobile    string          `json:"mobile"`
		AvatarURL string          `json:"avatar_url"`
		Status    json.RawMessage `json:"status"`
	}
	if err := json.Unmarshal(rawUser, &item); err != nil {
		return idp.DirectoryUser{}, fmt.Errorf("feishu user parse failed: %w", err)
	}
	rawPayload := rawUser
	if strings.TrimSpace(string(rawPayload)) == "" {
		rawPayload = data
	}
	return idp.DirectoryUser{
		ExternalUserID:  item.UserID,
		ExternalUnionID: item.UnionID,
		ExternalOpenID:  item.OpenID,
		Name:            item.Name,
		Email:           item.Email,
		Phone:           item.Mobile,
		AvatarURL:       item.AvatarURL,
		Status:          mapFeishuUserStatus(item.Status),
		RawProfile:      cloneBytes(rawPayload),
	}, nil
}

func (c *Client) getDepartmentByID(ctx context.Context, token string, departmentID string, idType string) (idp.DirectoryDepartment, error) {
	data, err := c.getObjectByID(ctx, token, "departments", departmentID, "department_id_type", idType)
	if err != nil {
		return idp.DirectoryDepartment{}, err
	}
	var envelope struct {
		Department json.RawMessage `json:"department"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return idp.DirectoryDepartment{}, err
	}
	rawDepartment := firstNonEmptyRaw(envelope.Department)
	if len(rawDepartment) == 0 {
		rawDepartment = cloneBytes(data)
	}
	var item struct {
		DepartmentID     string `json:"department_id"`
		OpenDepartmentID string `json:"open_department_id"`
		ParentID         string `json:"parent_department_id"`
		Name             string `json:"name"`
	}
	if err := json.Unmarshal(rawDepartment, &item); err != nil {
		return idp.DirectoryDepartment{}, fmt.Errorf("feishu department parse failed: %w", err)
	}
	externalID := firstNonEmpty(item.DepartmentID, item.OpenDepartmentID)
	return idp.DirectoryDepartment{
		ExternalDepartmentID:       externalID,
		ParentExternalDepartmentID: item.ParentID,
		Name:                       item.Name,
		RawProfile:                 cloneBytes(rawDepartment),
	}, nil
}

func (c *Client) getObjectByID(ctx context.Context, token string, objectType string, id string, typeParam string, idType string) ([]byte, error) {
	query := url.Values{}
	if idType != "" {
		query.Set(typeParam, idType)
	}
	path := "/open-apis/contact/v3/" + objectType + "/" + id
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var response struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, token, nil, &response); err != nil {
		return nil, err
	}
	if response.Code != 0 {
		return nil, fmt.Errorf("feishu %s get failed: code=%d msg=%s", objectType, response.Code, response.Msg)
	}
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("feishu %s get missing data", objectType)
	}
	return response.Data, nil
}

func (c *Client) doJSON(ctx context.Context, method string, path string, token string, body interface{}, out interface{}) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.cfg.BaseURL, "/")+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		message := strings.TrimSpace(string(body))
		if message == "" {
			return fmt.Errorf("feishu http status %d for %s", res.StatusCode, path)
		}
		return fmt.Errorf("feishu http status %d for %s: %s", res.StatusCode, path, message)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func mapFeishuUserStatus(rawStatus json.RawMessage) string {
	if len(bytes.TrimSpace(rawStatus)) == 0 {
		return "unknown"
	}
	var boolStatus struct {
		IsActivated bool `json:"is_activated"`
		IsFrozen    bool `json:"is_frozen"`
		IsResigned  bool `json:"is_resigned"`
	}
	if err := json.Unmarshal(rawStatus, &boolStatus); err == nil {
		if boolStatus.IsResigned || boolStatus.IsFrozen {
			return "disabled"
		}
		if boolStatus.IsActivated {
			return "active"
		}
		return "unknown"
	}
	var codeStatus struct {
		Status int `json:"status"`
	}
	if err := json.Unmarshal(rawStatus, &codeStatus); err == nil {
		if codeStatus.Status == 3 {
			return "disabled"
		}
		if codeStatus.Status == 1 || codeStatus.Status == 2 {
			return "active"
		}
	}
	var statusNum int
	if err := json.Unmarshal(rawStatus, &statusNum); err == nil {
		if statusNum == 3 {
			return "disabled"
		}
		if statusNum == 1 || statusNum == 2 {
			return "active"
		}
	}
	return "unknown"
}

func mapUserStatus(activated bool, frozen bool, resigned bool) string {
	if resigned || frozen {
		return "disabled"
	}
	if activated {
		return "active"
	}
	return "unknown"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func englishNameForUser(name string, values ...interface{}) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			parts = append(parts, typed)
		case json.RawMessage:
			parts = append(parts, englishNameFromI18n(typed))
		}
	}
	if english := normalizeLatinName(firstNonEmpty(parts...)); english != "" {
		return english
	}
	if !hasCJK(name) {
		return normalizeLatinName(name)
	}
	return pinyinDisplayName(name)
}

func englishNameFromI18n(raw json.RawMessage) string {
	if len(bytes.TrimSpace(raw)) == 0 {
		return ""
	}
	var names map[string]string
	if err := json.Unmarshal(raw, &names); err != nil {
		return ""
	}
	return firstNonEmpty(names["en_us"], names["en-US"], names["en"], names["en_US"])
}

func normalizeLatinName(value string) string {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return ""
	}
	for i, field := range fields {
		runes := []rune(field)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		fields[i] = string(runes)
	}
	return strings.Join(fields, "")
}

func hasCJK(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func pinyinDisplayName(value string) string {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	args.Fallback = func(r rune, _ pinyin.Args) []string {
		if unicode.IsSpace(r) {
			return nil
		}
		return []string{string(r)}
	}
	chunks := pinyin.Pinyin(value, args)
	parts := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		if normalized := normalizeLatinName(chunk[0]); normalized != "" {
			parts = append(parts, normalized)
		}
	}
	return strings.Join(parts, "")
}

func firstNonEmptyRaw(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(bytes.TrimSpace(value)) > 0 {
			out := make(json.RawMessage, len(value))
			copy(out, value)
			return out
		}
	}
	return nil
}

func cloneBytes(value []byte) []byte {
	out := make([]byte, len(value))
	copy(out, value)
	return out
}
