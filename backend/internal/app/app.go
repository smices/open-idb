// SPDX-License-Identifier: MIT

package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/smices/open-idb/internal/adminapi"
	"github.com/smices/open-idb/internal/audit"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/config"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/ephemeral"
	"github.com/smices/open-idb/internal/httpserver"
	"github.com/smices/open-idb/internal/idp"
	"github.com/smices/open-idb/internal/idp/feishu"
	"github.com/smices/open-idb/internal/platform/postgres"
	"github.com/smices/open-idb/internal/rbac"
	"github.com/smices/open-idb/internal/sso"
	"github.com/smices/open-idb/internal/worker"
	"go.uber.org/zap"
)

type App struct {
	cfg    config.Config
	logger *zap.Logger
	server *http.Server
	worker *worker.Worker
	close  func()
}

func New(ctx context.Context, cfg config.Config, logger *zap.Logger) (*App, error) {
	routerOptions := []httpserver.Option{}
	closeFn := func() {}
	var bgWorker *worker.Worker
	ephemeralStore := ephemeral.Store(ephemeral.NewMemoryStore())
	if cfg.RedisEnabled {
		redisStore, err := ephemeral.NewRedisStore(ctx, cfg.RedisURL)
		if err != nil {
			return nil, err
		}
		ephemeralStore = redisStore
		closeFn = func() {
			_ = ephemeralStore.Close()
		}
	}
	if cfg.DatabaseURL != "" {
		pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			_ = ephemeralStore.Close()
			return nil, err
		}
		previousCloseFn := closeFn
		closeFn = func() {
			auth.SetSessionResolver(nil)
			auth.SetAdminSessionResolver(nil)
			previousCloseFn()
			pool.Close()
		}
		queries := generated.New(pool)
		auth.SetSessionResolver(auth.NewDatabaseSessionResolver(queries))
		auth.SetAdminSessionResolver(auth.NewDatabaseAdminSessionResolver(pool))

		// Audit service is created early so it can be injected into all
		// handlers that need to write audit events.
		auditService := audit.NewService(queries)

		privateKey, err := sso.GenerateRSAKey()
		if err != nil {
			closeFn()
			return nil, err
		}
		service, err := sso.NewService(sso.ServiceConfig{
			Issuer:           cfg.OIDCIssuer,
			KeyID:            cfg.OIDCKeyID,
			PrivateKey:       privateKey,
			Store:            queries,
			TokenLookupStore: sso.NewDatabaseQuerier(queries),
			AuthCodeTTL:      cfg.AuthCodeTTL,
			AccessTokenTTL:   cfg.AccessTokenTTL,
			IDTokenTTL:       cfg.IDTokenTTL,
		})
		if err != nil {
			closeFn()
			return nil, err
		}
		handler := sso.NewHandler(service, auditService)
		handler.SetEphemeralStore(ephemeralStore)
		routerOptions = append(routerOptions, handler.RegisterRoutes)

		authService, err := auth.NewService(queries)
		if err != nil {
			closeFn()
			return nil, err
		}
		authHandler := auth.NewHandler(authService, auditService)
		authHandler.SetSessionTTL(cfg.SessionTTL)
		authHandler.SetEphemeralStore(ephemeralStore)
		routerOptions = append(routerOptions, authHandler.RegisterRoutes)

		adminAuthHandler := auth.NewAdminHandler(auth.NewAdminService(pool), auditService)
		adminAuthHandler.SetSessionTTL(cfg.SessionTTL)
		adminAuthHandler.SetEphemeralStore(ephemeralStore)
		routerOptions = append(routerOptions, adminAuthHandler.RegisterRoutes)

		// Console service (dashboard, current user, password)
		consoleService, err := adminapi.NewService(queries)
		if err != nil {
			closeFn()
			return nil, err
		}
		adminHandler := adminapi.NewHandler(nil, consoleService)
		routerOptions = append(routerOptions, adminHandler.RegisterRoutes)

		// Config service (IM providers)
		configService, err := adminapi.NewConfigService(queries, cfg.FeishuRedirectURI)
		if err != nil {
			closeFn()
			return nil, err
		}
		configHandler := adminapi.NewConfigHandler(configService)
		routerOptions = append(routerOptions, configHandler.RegisterRoutes)

		// Audit logging (read endpoint)
		auditHandler := audit.NewHandler(auditService)
		routerOptions = append(routerOptions, auditHandler.RegisterRoutes)

		// Admin CRUD APIs (with audit logging)
		adminCRUDService, err := adminapi.NewAdminService(queries, auditService)
		if err != nil {
			closeFn()
			return nil, err
		}
		adminCRUDService.SetTxStarter(pool)
		organizationTreeCache := adminapi.NewOrganizationTreeCache(ephemeralStore)
		adminCRUDService.SetOrganizationTreeCache(organizationTreeCache)
		entityHandler := adminapi.NewEntityHandler(adminCRUDService)
		routerOptions = append(routerOptions, entityHandler.RegisterRoutes)
		platformHandler := adminapi.NewPlatformHandler(adminCRUDService)
		routerOptions = append(routerOptions, platformHandler.RegisterRoutes)
		userHandler := adminapi.NewUserHandler(adminCRUDService)
		routerOptions = append(routerOptions, userHandler.RegisterRoutes)
		directoryHandler := adminapi.NewDirectoryHandler(adminCRUDService)
		routerOptions = append(routerOptions, directoryHandler.RegisterRoutes)
		applicationHandler := adminapi.NewApplicationHandler(adminCRUDService)
		routerOptions = append(routerOptions, applicationHandler.RegisterRoutes)
		roleHandler := adminapi.NewRoleHandler(adminCRUDService)
		routerOptions = append(routerOptions, roleHandler.RegisterRoutes)
		permissionHandler := adminapi.NewPermissionHandler(adminCRUDService)
		routerOptions = append(routerOptions, permissionHandler.RegisterRoutes)
		syncJobHandler := adminapi.NewSyncJobHandler(adminCRUDService)
		routerOptions = append(routerOptions, syncJobHandler.RegisterRoutes)

		// Session management (list/revoke)
		sessionHandler := adminapi.NewSessionHandler(adminCRUDService)
		routerOptions = append(routerOptions, sessionHandler.RegisterRoutes)

		// Identity Sources CRUD
		identitySourceHandler := adminapi.NewIdentitySourceHandler(adminCRUDService)
		routerOptions = append(routerOptions, identitySourceHandler.RegisterRoutes)

		// Account Bindings
		bindingHandler := adminapi.NewBindingHandler(adminCRUDService)
		routerOptions = append(routerOptions, bindingHandler.RegisterRoutes)

		// Organization tree read model
		organizationHandler := adminapi.NewOrganizationHandler(adminCRUDService)
		routerOptions = append(routerOptions, organizationHandler.RegisterRoutes)
		oidcDirectoryHandler := adminapi.NewOIDCDirectoryHandler(adminCRUDService, service)
		routerOptions = append(routerOptions, oidcDirectoryHandler.RegisterRoutes)

		// OIDC Clients CRUD
		oidcClientHandler := adminapi.NewOIDCClientHandler(adminCRUDService)
		routerOptions = append(routerOptions, oidcClientHandler.RegisterRoutes)

		// Internal v1 APIs (service-to-service authorization)
		internalService, err := adminapi.NewInternalService(queries)
		if err != nil {
			closeFn()
			return nil, err
		}
		internalHandler := adminapi.NewInternalHandler(internalService)
		routerOptions = append(routerOptions, internalHandler.RegisterRoutes)

		// RBAC with Casbin
		enforcer, err := rbac.NewEnforcer()
		if err != nil {
			closeFn()
			return nil, err
		}
		rbacService, err := rbac.NewService(queries, enforcer)
		if err != nil {
			closeFn()
			return nil, err
		}
		rbacHandler := rbac.NewHandler(rbacService)
		routerOptions = append(routerOptions, rbacHandler.RegisterRoutes)

		// Login providers discovery endpoint + Feishu login and sync.
		providerService := auth.NewLoginProviderService(queries, cfg.FeishuAppID, cfg.FeishuAppSecret, cfg.FeishuRedirectURI)
		feishuLoginService := auth.NewFeishuLoginService(queries, nil, cfg.SessionTTL)
		feishuLoginService.SetClientResolver(feishuClientResolver{
			buildClient: func(ctx context.Context, entityID string) (*feishu.Client, error) {
				resolvedCfg, err := providerService.ResolveFeishuConfig(ctx, entityID)
				if err != nil {
					return nil, err
				}
				return feishu.NewClient(feishu.Config{
					AppID:     resolvedCfg.AppID,
					AppSecret: resolvedCfg.AppSecret,
					BaseURL:   cfg.FeishuBaseURL,
				}, nil)
			},
			buildWorkplaceClient: func(ctx context.Context, entityID string, clientID string) (*feishu.Client, error) {
				resolvedCfg, err := providerService.ResolveFeishuWorkplaceConfig(ctx, entityID, clientID)
				if err != nil {
					return nil, err
				}
				return feishu.NewClient(feishu.Config{
					AppID:     resolvedCfg.AppID,
					AppSecret: resolvedCfg.AppSecret,
					BaseURL:   cfg.FeishuBaseURL,
				}, nil)
			},
		})

		syncService, err := idp.NewSyncService(idp.SyncServiceConfig{
			Queries: queries,
			ProviderFactory: func(ctx context.Context, entityID, sourceID, provider string) (idp.DirectoryProvider, error) {
				switch strings.TrimSpace(provider) {
				case "", "feishu":
					resolvedCfg, err := providerService.ResolveFeishuConfig(ctx, entityID)
					if err != nil {
						return nil, err
					}
					return feishu.NewClient(feishu.Config{
						AppID:     resolvedCfg.AppID,
						AppSecret: resolvedCfg.AppSecret,
						BaseURL:   cfg.FeishuBaseURL,
					}, nil)
				case "dingtalk", "wecom", "ldap", "local":
					return nil, fmt.Errorf("directory provider %s is not implemented for sync", provider)
				default:
					return nil, fmt.Errorf("unsupported directory provider: %s", provider)
				}
			},
			Audit: auditService,
		})
		if err != nil {
			closeFn()
			return nil, err
		}
		syncService.SetTxStarter(pool)
		syncHandler := adminapi.NewHandler(syncService, nil)
		routerOptions = append(routerOptions, syncHandler.RegisterRoutes)

		// Create background worker for sync job processing
		syncRunner := worker.NewSyncRunner(syncService, logger)
		syncRunner.SetOrganizationTreeCacheInvalidator(organizationTreeCache)
		cleanupRunner := worker.NewCleanupRunner(queries, time.Hour, logger)
		bgWorker = worker.New(worker.Config{}, logger, syncRunner, auditService, cleanupRunner)

		feishuLoginHandler := auth.NewFeishuLoginHandler(feishuLoginService, providerService, cfg.FeishuAppID, cfg.FeishuRedirectURI, auditService)
		feishuLoginHandler.SetWebBaseURL(cfg.WebBaseURL)
		feishuLoginHandler.SetEphemeralStore(ephemeralStore)
		feishuLoginHandler.SetLogger(logger)
		routerOptions = append(routerOptions, feishuLoginHandler.RegisterRoutes)
	}

	return &App{
		cfg:    cfg,
		logger: logger,
		server: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           httpserver.NewRouter(routerOptions...),
			ReadHeaderTimeout: 5 * time.Second,
		},
		worker: bgWorker,
		close:  closeFn,
	}, nil
}

type feishuClientResolver struct {
	buildClient          func(ctx context.Context, entityID string) (*feishu.Client, error)
	buildWorkplaceClient func(ctx context.Context, entityID string, clientID string) (*feishu.Client, error)
}

func (r feishuClientResolver) GetFeishuUserProvider(ctx context.Context, entityID string, _ string) (auth.FeishuUserProvider, error) {
	client, err := r.buildClient(ctx, entityID)
	if err != nil {
		return nil, err
	}
	return feishuClientAdapter{client}, nil
}

func (r feishuClientResolver) GetFeishuWorkplaceUserProvider(ctx context.Context, entityID string, _ string, clientID string) (auth.FeishuUserProvider, error) {
	if r.buildWorkplaceClient != nil {
		client, err := r.buildWorkplaceClient(ctx, entityID, clientID)
		if err != nil {
			return nil, err
		}
		return feishuClientAdapter{client}, nil
	}
	client, err := r.buildClient(ctx, entityID)
	if err != nil {
		return nil, err
	}
	return feishuClientAdapter{client}, nil
}

// feishuClientAdapter bridges feishu.Client to auth.FeishuUserProvider.
type feishuClientAdapter struct {
	client *feishu.Client
}

func (a feishuClientAdapter) GetUserInfoByCode(ctx context.Context, code string) (auth.FeishuUserInfo, error) {
	r, err := a.client.GetUserInfoByCode(ctx, code)
	if err != nil {
		return auth.FeishuUserInfo{}, err
	}
	return auth.FeishuUserInfo{
		UserID: r.UserID, UnionID: r.UnionID, OpenID: r.OpenID,
		Name: r.Name, Email: r.Email, Phone: r.Phone,
		AvatarURL: r.AvatarURL, Status: r.Status, RawProfile: r.RawProfile,
	}, nil
}

func (a feishuClientAdapter) GetUserInfoByAppCode(ctx context.Context, authCode string) (auth.FeishuUserInfo, error) {
	r, err := a.client.GetUserInfoByAppCode(ctx, authCode)
	if err != nil {
		return auth.FeishuUserInfo{}, err
	}
	return auth.FeishuUserInfo{
		UserID: r.UserID, UnionID: r.UnionID, OpenID: r.OpenID,
		Name: r.Name, Email: r.Email, Phone: r.Phone,
		AvatarURL: r.AvatarURL, Status: r.Status, RawProfile: r.RawProfile,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", a.cfg.HTTPAddr)
	if err != nil {
		return err
	}
	return a.serve(ctx, listener)
}

func (a *App) serve(ctx context.Context, listener net.Listener) error {
	defer a.close()

	// Start background worker if configured (sync jobs, async audit)
	if a.worker != nil {
		a.worker.Start(ctx)
	}

	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("starting http server", zap.String("addr", a.cfg.HTTPAddr))
		if err := a.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		// Stop worker first to drain async audit events
		if a.worker != nil {
			a.worker.Stop()
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if a.worker != nil {
			a.worker.Stop()
		}
		return err
	}
}
