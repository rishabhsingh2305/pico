package routestest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"picoservice/internal/authz"
	"picoservice/internal/entities"
	"picoservice/internal/routes"
	"picoservice/internal/services"
	"picoservice/test/fakes"
)

const flowSecret = "01234567890123456789012345678901"

// TestAuthFlowWithFeatureRoles logs in with a real AuthService, calls the
// router with the real access token, changes the membership roles, refreshes,
// and proves the new permissions take effect without restarting the app.
func TestAuthFlowWithFeatureRoles(t *testing.T) {
	policy, err := authz.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	user := entities.User{ID: "user-1", Email: "ada@example.com", EmailVerified: true}
	users := &flowUserStore{user: user}
	memberships := fakes.NewMembershipDAO()
	memberships.Add("org-1", user.ID, map[string]string{
		entities.FeatureProject: entities.FeatureRoleEditor,
		entities.FeatureExpense: entities.FeatureRoleViewer,
	})
	orgs := &flowOrgs{orgID: "org-1", ownerID: "owner-1"}

	membershipService := services.NewMembershipService(memberships, orgs, users, policy)
	roleResolver := services.NewOrgRoleResolver(memberships, orgs)
	authService, err := services.NewAuthService(users, plainHasher{}, flowSecret, roleResolver)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}

	fakeServices := fakes.NewServices()
	r, err := routes.New(routes.Deps{
		Logger:         testLogger(),
		Pinger:         stubPinger{},
		Users:          fakeServices.Users,
		Auth:           authService,
		Orgs:           fakeServices.Orgs,
		Memberships:    membershipService,
		Projects:       fakeServices.Projects,
		Expenses:       fakeServices.Expenses,
		Credentials:    fakeServices.Credentials,
		Policy:         policy,
		AllowedOrigins: []string{"http://localhost:5173"},
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	login := postJSON(t, r, "/auth/login", `{"email":"ada@example.com","password":"correct horse"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", login.Code, login.Body.String())
	}
	var session struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	assertStatus(t, get(t, r, "/projects?orgId=org-1", session.AccessToken), http.StatusOK)
	assertStatus(t, postJSON(t, r, "/projects?orgId=org-1", `{"name":"Launch","currency":"USD"}`, session.AccessToken), http.StatusCreated)
	assertStatus(t, get(t, r, "/expenses?orgId=org-1", session.AccessToken), http.StatusOK)
	assertStatus(t, postJSON(t, r, "/expenses?orgId=org-1", `{"projectId":"project-1","amount":100,"spentAt":"2026-01-02T03:04:05Z"}`, session.AccessToken), http.StatusForbidden)

	memberships.SetRoles("org-1", user.ID, map[string]string{
		entities.FeatureProject: entities.FeatureRoleViewer,
		entities.FeatureExpense: entities.FeatureRoleAdmin,
	})

	refresh := postJSON(t, r, "/auth/refresh", `{"refreshToken":"`+session.RefreshToken+`"}`, "")
	if refresh.Code != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d: %s", refresh.Code, refresh.Body.String())
	}
	var refreshed struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(refresh.Body.Bytes(), &refreshed); err != nil {
		t.Fatalf("decode refresh: %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.AccessToken == session.AccessToken {
		t.Fatal("expected a new access token")
	}

	assertStatus(t, postJSON(t, r, "/expenses?orgId=org-1", `{"projectId":"project-1","amount":100,"spentAt":"2026-01-02T03:04:05Z"}`, refreshed.AccessToken), http.StatusCreated)
	assertStatus(t, postJSON(t, r, "/projects?orgId=org-1", `{"name":"Launch","currency":"USD"}`, refreshed.AccessToken), http.StatusForbidden)
	assertStatus(t, get(t, r, "/projects?orgId=org-1", refreshed.AccessToken), http.StatusOK)
}

func get(t *testing.T, r http.Handler, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func postJSON(t *testing.T, r http.Handler, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("expected %d, got %d: %s", want, rec.Code, rec.Body.String())
	}
}

type plainHasher struct{}

func (plainHasher) Hash(password string) (string, error) {
	return "plain:" + password, nil
}

func (plainHasher) Compare(hash, password string) error {
	if hash == "plain:"+password {
		return nil
	}
	return entities.ErrPasswordMismatch
}

type flowUserStore struct {
	user entities.User
}

func (s *flowUserStore) FindByEmail(_ context.Context, email string) (entities.User, error) {
	if strings.EqualFold(email, s.user.Email) {
		return s.user, nil
	}
	return entities.User{}, entities.ErrUserNotFound
}

func (s *flowUserStore) FindByID(_ context.Context, id string) (entities.User, error) {
	if id == s.user.ID {
		return s.user, nil
	}
	return entities.User{}, entities.ErrUserNotFound
}

func (s *flowUserStore) PasswordHash(_ context.Context, id string) (string, error) {
	if id != s.user.ID {
		return "", entities.ErrUserNotFound
	}
	return "plain:correct horse", nil
}

func (s *flowUserStore) Exists(_ context.Context, id string) (bool, error) {
	return id == s.user.ID, nil
}

type flowOrgs struct {
	orgID   string
	ownerID string
}

func (s *flowOrgs) OwnerID(_ context.Context, orgID string) (string, bool, error) {
	if orgID == s.orgID {
		return s.ownerID, true, nil
	}
	return "", false, nil
}

func (s *flowOrgs) OwnerIDs(_ context.Context, orgIDs []string) (map[string]string, error) {
	owners := make(map[string]string, len(orgIDs))
	for _, orgID := range orgIDs {
		if orgID == s.orgID {
			owners[orgID] = s.ownerID
		}
	}
	return owners, nil
}
