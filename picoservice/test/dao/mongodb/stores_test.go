package mongodbtest

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"picoservice/internal/dao"
	"picoservice/internal/dao/mongodb"
	"picoservice/internal/entities"
)

var integrationCollections = []string{"users", "orgs", "memberships", "projects", "expenses"}

func TestStoresIntegration(t *testing.T) {
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("set MONGODB_TEST_URI to run MongoDB integration tests")
	}

	database := os.Getenv("MONGODB_TEST_DATABASE")
	if database == "" {
		database = "picoservice_test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	raw, err := drivermongo.Connect(options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(5 * time.Second).
		SetConnectTimeout(5 * time.Second))
	if err != nil {
		t.Fatalf("connect raw client: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := clearCollections(cleanupCtx, raw, database); err != nil {
			t.Errorf("clear test collections: %v", err)
		}
		if err := raw.Disconnect(cleanupCtx); err != nil {
			t.Errorf("disconnect raw client: %v", err)
		}
	})

	if err := clearCollections(ctx, raw, database); err != nil {
		t.Fatalf("clear test collections: %v", err)
	}

	client, err := mongodb.New(mongodb.Config{URI: uri, Database: database})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := client.Close(cleanupCtx); err != nil {
			t.Errorf("close client: %v", err)
		}
	})

	if err := client.EnsureIndexes(ctx); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	if err := client.EnsureIndexes(ctx); err != nil {
		t.Fatalf("ensure indexes twice: %v", err)
	}

	now := time.Now().UTC()

	userStore := mongodb.NewUserStore(client)
	user, err := userStore.Insert(ctx, entities.User{
		Name:      "Ada",
		Email:     "ada@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}, "password-hash")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if _, err := userStore.Insert(ctx, entities.User{
		Name:      "Duplicate",
		Email:     "ada@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}, "password-hash"); !errors.Is(err, entities.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}

	hash, err := userStore.PasswordHash(ctx, user.ID)
	if err != nil {
		t.Fatalf("get password hash: %v", err)
	}
	if got, want := hash, "password-hash"; got != want {
		t.Fatalf("expected hash %q, got %q", want, got)
	}

	foundByEmail, err := userStore.FindByEmail(ctx, "ada@example.com")
	if err != nil {
		t.Fatalf("find user by email: %v", err)
	}
	if foundByEmail.ID != user.ID {
		t.Fatalf("expected user %q, got %q", user.ID, foundByEmail.ID)
	}

	verified, err := userStore.MarkEmailVerified(ctx, user.ID, user.Email, now)
	if err != nil {
		t.Fatalf("mark email verified: %v", err)
	}
	if !verified.EmailVerified || verified.EmailVerifiedAt == nil {
		t.Fatalf("expected verified user, got %+v", verified)
	}

	sentAt := now.Add(time.Minute)
	if err := userStore.SetVerificationSentAt(ctx, user.ID, sentAt); err != nil {
		t.Fatalf("set verification sent time: %v", err)
	}
	reloaded, err := userStore.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.VerificationSentAt == nil || !reloaded.VerificationSentAt.Equal(sentAt) {
		t.Fatalf("expected verification sent time %v, got %v", sentAt, reloaded.VerificationSentAt)
	}

	updatedUser, err := userStore.Update(ctx, user.ID, "Ada Lovelace", user.Email, false, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("update user: %v", err)
	}
	if updatedUser.EmailVerified || updatedUser.EmailVerifiedAt != nil {
		t.Fatalf("expected verification to be cleared, got %+v", updatedUser)
	}

	temporary, err := userStore.Insert(ctx, entities.User{
		Name:      "Temporary",
		Email:     "temporary@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}, "password-hash")
	if err != nil {
		t.Fatalf("insert temporary user: %v", err)
	}
	if err := userStore.Delete(ctx, temporary.ID); err != nil {
		t.Fatalf("delete temporary user: %v", err)
	}
	if _, err := userStore.FindByID(ctx, temporary.ID); !errors.Is(err, entities.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}

	orgStore := mongodb.NewOrgStore(client)
	org, err := orgStore.Insert(ctx, entities.Org{
		Name:      "Acme",
		UserID:    user.ID,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	ownerID, found, err := orgStore.OwnerID(ctx, org.ID)
	if err != nil {
		t.Fatalf("get org owner: %v", err)
	}
	if !found || ownerID != user.ID {
		t.Fatalf("expected owner %q, got %q (found %t)", user.ID, ownerID, found)
	}

	membershipStore := mongodb.NewMembershipStore(client)
	if err := membershipStore.AddOwner(ctx, org.ID, user.ID); err != nil {
		t.Fatalf("add owner membership: %v", err)
	}
	if err := membershipStore.AddOwner(ctx, org.ID, user.ID); !errors.Is(err, entities.ErrAlreadyMember) {
		t.Fatalf("expected ErrAlreadyMember, got %v", err)
	}

	orgIDs, err := membershipStore.ListOrgIDsByUser(ctx, user.ID, 20, 0)
	if err != nil {
		t.Fatalf("list user orgs: %v", err)
	}
	if got, want := len(orgIDs), 1; got != want {
		t.Fatalf("expected %d org, got %d", want, got)
	}
	if orgIDs[0] != org.ID {
		t.Fatalf("expected org %q, got %q", org.ID, orgIDs[0])
	}

	projectStore := mongodb.NewProjectStore(client)
	project, err := projectStore.Insert(ctx, entities.Project{
		OrgID:     org.ID,
		UserID:    user.ID,
		Name:      "Launch",
		Currency:  "USD",
		Tags:      []string{"food", "travel"},
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if got, want := project.Currency, "USD"; got != want {
		t.Fatalf("expected project currency %q, got %q", want, got)
	}
	if got, want := project.Tags, []string{"food", "travel"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected tags %v, got %v", want, got)
	}
	projectsList, err := projectStore.List(ctx, dao.ProjectFilter{
		OrgID: org.ID,
		Tags:  []string{"food", "other"},
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if got, want := len(projectsList), 1; got != want {
		t.Fatalf("expected %d project, got %d", want, got)
	}
	noProjects, err := projectStore.List(ctx, dao.ProjectFilter{
		OrgID: org.ID,
		Tags:  []string{"other"},
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("list projects without matching tag: %v", err)
	}
	if got, want := len(noProjects), 0; got != want {
		t.Fatalf("expected %d project, got %d", want, got)
	}

	newTags := []string{"travel"}
	updatedProject, err := projectStore.Update(ctx, org.ID, project.ID, dao.ProjectUpdate{
		Tags:      &newTags,
		UpdatedAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("update project tags: %v", err)
	}
	if got, want := updatedProject.Tags, []string{"travel"}; len(got) != 1 || got[0] != want[0] {
		t.Fatalf("expected tags %v, got %v", want, got)
	}

	expenseStore := mongodb.NewExpenseStore(client)
	expense, err := expenseStore.Insert(ctx, entities.Expense{
		OrgID:     org.ID,
		UserID:    user.ID,
		ProjectID: project.ID,
		Type:      entities.ExpenseTypeDebit,
		Amount:    1234,
		Currency:  "USD",
		Tags:      []string{"food"},
		SpentAt:   now,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("insert expense: %v", err)
	}
	if _, err := expenseStore.Insert(ctx, entities.Expense{
		OrgID:     org.ID,
		UserID:    user.ID,
		ProjectID: project.ID,
		Type:      entities.ExpenseTypeCredit,
		Amount:    5000,
		Currency:  "USD",
		Tags:      []string{"income"},
		SpentAt:   now,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("insert credit: %v", err)
	}
	if _, err := raw.Database(database).Collection("expenses").InsertOne(ctx, bson.M{
		"_id":         bson.NewObjectID(),
		"orgId":       org.ID,
		"userId":      user.ID,
		"projectId":   project.ID,
		"amount":      int64(700),
		"currency":    "USD",
		"description": "Legacy",
		"tags":        []string{"legacy"},
		"spentAt":     now,
		"createdAt":   now,
		"updatedAt":   now,
	}); err != nil {
		t.Fatalf("insert legacy expense: %v", err)
	}

	expenseList, err := expenseStore.List(ctx, dao.ExpenseFilter{
		OrgID:     org.ID,
		ProjectID: project.ID,
		Tags:      []string{"food"},
		Limit:     20,
	})
	if err != nil {
		t.Fatalf("list expenses: %v", err)
	}
	if got, want := len(expenseList), 1; got != want {
		t.Fatalf("expected %d expense, got %d", want, got)
	}
	if got, want := expenseList[0].Type, entities.ExpenseTypeDebit; got != want {
		t.Fatalf("expected type %q, got %q", want, got)
	}

	summary, err := expenseStore.Summarize(ctx, dao.ExpenseFilter{
		OrgID:     org.ID,
		ProjectID: project.ID,
		Tags:      []string{"food"},
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("summarize expenses: %v", err)
	}
	if got, want := summary.Count, int64(1); got != want {
		t.Fatalf("expected summary count %d, got %d", want, got)
	}
	if got, want := summary.DebitAmount, int64(1234); got != want {
		t.Fatalf("expected summary debit %d, got %d", want, got)
	}
	if got, want := summary.CreditAmount, int64(0); got != want {
		t.Fatalf("expected summary credit %d, got %d", want, got)
	}

	allSummary, err := expenseStore.Summarize(ctx, dao.ExpenseFilter{
		OrgID:     org.ID,
		ProjectID: project.ID,
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("summarize all expenses: %v", err)
	}
	if got, want := allSummary.Count, int64(3); got != want {
		t.Fatalf("expected all summary count %d, got %d", want, got)
	}
	if got, want := allSummary.DebitAmount, int64(1934); got != want {
		t.Fatalf("expected all summary debit %d, got %d", want, got)
	}
	if got, want := allSummary.CreditAmount, int64(5000); got != want {
		t.Fatalf("expected all summary credit %d, got %d", want, got)
	}

	amount := int64(4321)
	expenseType := entities.ExpenseTypeCredit
	updated, err := expenseStore.Update(ctx, org.ID, expense.ID, dao.ExpenseUpdate{
		Type:      &expenseType,
		Amount:    &amount,
		UpdatedAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("update expense: %v", err)
	}
	if got, want := updated.Amount, amount; got != want {
		t.Fatalf("expected amount %d, got %d", want, got)
	}
	if got, want := updated.Type, expenseType; got != want {
		t.Fatalf("expected type %q, got %q", want, got)
	}

	if err := expenseStore.Delete(ctx, org.ID, expense.ID); err != nil {
		t.Fatalf("delete expense: %v", err)
	}
	if _, err := expenseStore.FindByID(ctx, org.ID, expense.ID); !errors.Is(err, entities.ErrExpenseNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func clearCollections(ctx context.Context, client *drivermongo.Client, database string) error {
	for _, collection := range integrationCollections {
		if _, err := client.Database(database).Collection(collection).DeleteMany(ctx, bson.D{}); err != nil {
			return err
		}
	}
	return nil
}
