// SPDX-License-Identifier: MIT

package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// UserInfoResult represents a Feishu user obtained via OAuth.
type UserInfoResult struct {
	UserID     string
	UnionID    string
	OpenID     string
	Name       string
	Email      string
	Phone      string
	AvatarURL  string
	Status     string // "active" or "disabled"
	RawProfile []byte
}

// appAccessToken fetches an app-level access token using app credentials.
func (c *Client) appAccessToken(ctx context.Context) (string, error) {
	payload := map[string]string{
		"app_id":     c.cfg.AppID,
		"app_secret": c.cfg.AppSecret,
	}
	var response struct {
		Code           int    `json:"code"`
		Msg            string `json:"msg"`
		AppAccessToken string `json:"app_access_token"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/open-apis/auth/v3/app_access_token/internal", "", payload, &response); err != nil {
		return "", err
	}
	if response.Code != 0 {
		return "", fmt.Errorf("feishu app token failed: code=%d msg=%s", response.Code, response.Msg)
	}
	if response.AppAccessToken == "" {
		return "", fmt.Errorf("feishu app token response missing token")
	}
	return response.AppAccessToken, nil
}

// GetUserInfoByCode exchanges an OAuth authorization code for user info.
// Flow: app_access_token -> user_access_token (OIDC path) -> user info.
func (c *Client) GetUserInfoByCode(ctx context.Context, code string) (UserInfoResult, error) {
	appToken, err := c.appAccessToken(ctx)
	if err != nil {
		return UserInfoResult{}, err
	}

	// Exchange code for user_access_token via OIDC endpoint.
	payload := map[string]string{
		"grant_type": "authorization_code",
		"code":       code,
	}
	var tokenResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/open-apis/authen/v1/oidc/access_token", appToken, payload, &tokenResp); err != nil {
		return UserInfoResult{}, err
	}
	if tokenResp.Code != 0 {
		return UserInfoResult{}, fmt.Errorf("feishu oidc token exchange failed: code=%d msg=%s", tokenResp.Code, tokenResp.Msg)
	}
	if tokenResp.Data.AccessToken == "" {
		return UserInfoResult{}, fmt.Errorf("feishu oidc token response missing access_token")
	}

	return c.fetchUserInfo(ctx, tokenResp.Data.AccessToken)
}

// GetUserInfoByAppCode exchanges a Feishu app auth code (for embedded apps) for user info.
// Uses the non-OIDC /open-apis/authen/v1/access_token path.
func (c *Client) GetUserInfoByAppCode(ctx context.Context, authCode string) (UserInfoResult, error) {
	appToken, err := c.appAccessToken(ctx)
	if err != nil {
		return UserInfoResult{}, err
	}

	// Exchange auth code for user_access_token via non-OIDC endpoint.
	payload := map[string]string{
		"app_access_token": appToken,
		"grant_type":       "authorization_code",
		"code":             authCode,
	}
	var tokenResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/open-apis/authen/v1/access_token", "", payload, &tokenResp); err != nil {
		return UserInfoResult{}, err
	}
	if tokenResp.Code != 0 {
		return UserInfoResult{}, fmt.Errorf("feishu app token exchange failed: code=%d msg=%s", tokenResp.Code, tokenResp.Msg)
	}
	if tokenResp.Data.AccessToken == "" {
		return UserInfoResult{}, fmt.Errorf("feishu app token response missing access_token")
	}

	return c.fetchUserInfo(ctx, tokenResp.Data.AccessToken)
}

// fetchUserInfo retrieves user profile using a user_access_token.
func (c *Client) fetchUserInfo(ctx context.Context, userAccessToken string) (UserInfoResult, error) {
	var response struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/open-apis/authen/v1/user_info", userAccessToken, nil, &response); err != nil {
		return UserInfoResult{}, err
	}
	if response.Code != 0 {
		return UserInfoResult{}, fmt.Errorf("feishu user info failed: code=%d msg=%s", response.Code, response.Msg)
	}

	var profile struct {
		OpenID      string `json:"open_id"`
		UnionID     string `json:"union_id"`
		UserID      string `json:"user_id"`
		Name        string `json:"name"`
		Email       string `json:"email"`
		Mobile      string `json:"mobile"`
		AvatarURL   string `json:"avatar_url"`
		EntityKey   string `json:"entity_key"`
		IsActivated *bool  `json:"is_activated,omitempty"`
		IsFrozen    bool   `json:"is_frozen,omitempty"`
		IsResigned  bool   `json:"is_resigned,omitempty"`
	}
	if err := json.Unmarshal(response.Data, &profile); err != nil {
		return UserInfoResult{}, fmt.Errorf("feishu user info parse failed: %w", err)
	}

	// Determine status from optional fields; default to "active" since user just authenticated.
	status := "active"
	if profile.IsResigned || profile.IsFrozen {
		status = "disabled"
	} else if profile.IsActivated != nil {
		if *profile.IsActivated {
			status = "active"
		} else {
			status = "disabled"
		}
	}

	// Prefer enterprise user_id; fall back to open_id which is always present.
	userID := profile.UserID
	if userID == "" {
		userID = profile.OpenID
	}

	return UserInfoResult{
		UserID:     userID,
		UnionID:    profile.UnionID,
		OpenID:     profile.OpenID,
		Name:       profile.Name,
		Email:      profile.Email,
		Phone:      profile.Mobile,
		AvatarURL:  profile.AvatarURL,
		Status:     status,
		RawProfile: cloneBytes([]byte(response.Data)),
	}, nil
}
