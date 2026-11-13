package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
)

const expensesCollection = "expenses"

// ExpenseStore persists expenses in MongoDB.
type ExpenseStore struct {
	collection *mongo.Collection
}

var _ dao.ExpenseDAO = (*ExpenseStore)(nil)

// NewExpenseStore returns an expense store backed by client.
func NewExpenseStore(client *Client) *ExpenseStore {
	return &ExpenseStore{collection: client.collection(expensesCollection)}
}

type expenseDocument struct {
	ID          bson.ObjectID        `bson:"_id"`
	OrgID       string               `bson:"orgId"`
	UserID      string               `bson:"userId"`
	ProjectID   string               `bson:"projectId"`
	Type        entities.ExpenseType `bson:"type"`
	Amount      int64                `bson:"amount"`
	Currency    string               `bson:"currency"`
	Description string               `bson:"description"`
	Tags        []string             `bson:"tags"`
	SpentAt     time.Time            `bson:"spentAt"`
	CreatedAt   time.Time            `bson:"createdAt"`
	UpdatedAt   time.Time            `bson:"updatedAt"`
}

func (d expenseDocument) expense() entities.Expense {
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}

	return entities.Expense{
		ID:          d.ID.Hex(),
		OrgID:       d.OrgID,
		UserID:      d.UserID,
		ProjectID:   d.ProjectID,
		Type:        d.Type.OrDefault(),
		Amount:      d.Amount,
		Currency:    d.Currency,
		Description: d.Description,
		Tags:        tags,
		SpentAt:     d.SpentAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// Insert stores an expense and returns it with its generated ID.
func (s *ExpenseStore) Insert(ctx context.Context, expense entities.Expense) (entities.Expense, error) {
	tags := expense.Tags
	if tags == nil {
		tags = []string{}
	}

	document := expenseDocument{
		ID:          bson.NewObjectID(),
		OrgID:       expense.OrgID,
		UserID:      expense.UserID,
		ProjectID:   expense.ProjectID,
		Type:        expense.Type.OrDefault(),
		Amount:      expense.Amount,
		Currency:    expense.Currency,
		Description: expense.Description,
		Tags:        tags,
		SpentAt:     expense.SpentAt,
		CreatedAt:   expense.CreatedAt,
		UpdatedAt:   expense.UpdatedAt,
	}

	if _, err := s.collection.InsertOne(ctx, document); err != nil {
		return entities.Expense{}, fmt.Errorf("insert expense: %w", err)
	}
	return document.expense(), nil
}

// FindByID returns an expense scoped to orgID.
func (s *ExpenseStore) FindByID(ctx context.Context, orgID, id string) (entities.Expense, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.Expense{}, entities.ErrExpenseNotFound
	}

	var document expenseDocument
	err = s.collection.FindOne(ctx, bson.M{"_id": objectID, "orgId": orgID}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Expense{}, entities.ErrExpenseNotFound
	}
	if err != nil {
		return entities.Expense{}, fmt.Errorf("find expense: %w", err)
	}
	return document.expense(), nil
}

// List returns expenses for an organization ordered by spent-at time.
func (s *ExpenseStore) List(ctx context.Context, in dao.ExpenseFilter) ([]entities.Expense, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "spentAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(in.Limit)).
		SetSkip(int64(in.Offset))

	cursor, err := s.collection.Find(ctx, expenseMatch(in), opts)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}

	var documents []expenseDocument
	if err := cursor.All(ctx, &documents); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode expenses: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close expense cursor: %w", err)
	}

	items := make([]entities.Expense, 0, len(documents))
	for _, document := range documents {
		items = append(items, document.expense())
	}
	return items, nil
}

// SummarizeByProject returns count and amount totals grouped by project and
// currency for every expense matching the filter. Pagination is ignored.
func (s *ExpenseStore) SummarizeByProject(ctx context.Context, in dao.ExpenseFilter) ([]dao.ExpenseSummaryGroup, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: expenseMatch(in)}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"projectId": "$projectId",
				"currency":  bson.M{"$ifNull": []any{"$currency", "USD"}},
			},
			"count": bson.M{"$sum": 1},
			"debit": bson.M{"$sum": bson.M{"$cond": []any{
				bson.M{"$eq": []any{
					bson.M{"$ifNull": []any{"$type", string(entities.ExpenseTypeDebit)}},
					string(entities.ExpenseTypeDebit),
				}},
				"$amount",
				0,
			}}},
			"credit": bson.M{"$sum": bson.M{"$cond": []any{
				bson.M{"$eq": []any{
					bson.M{"$ifNull": []any{"$type", string(entities.ExpenseTypeDebit)}},
					string(entities.ExpenseTypeCredit),
				}},
				"$amount",
				0,
			}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{
			{Key: "_id.projectId", Value: 1},
			{Key: "_id.currency", Value: 1},
		}}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("summarize expenses: %w", err)
	}

	var results []struct {
		ID struct {
			ProjectID string `bson:"projectId"`
			Currency  string `bson:"currency"`
		} `bson:"_id"`
		Count  int64 `bson:"count"`
		Debit  int64 `bson:"debit"`
		Credit int64 `bson:"credit"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		_ = cursor.Close(ctx)
		return nil, fmt.Errorf("decode expense summary: %w", err)
	}
	if err := cursor.Close(ctx); err != nil {
		return nil, fmt.Errorf("close expense summary cursor: %w", err)
	}

	groups := make([]dao.ExpenseSummaryGroup, 0, len(results))
	for _, result := range results {
		groups = append(groups, dao.ExpenseSummaryGroup{
			ProjectID:    result.ID.ProjectID,
			Currency:     result.ID.Currency,
			Count:        result.Count,
			DebitAmount:  result.Debit,
			CreditAmount: result.Credit,
		})
	}
	return groups, nil
}

func expenseMatch(in dao.ExpenseFilter) bson.M {
	filter := bson.M{"orgId": in.OrgID}

	if len(in.ProjectIDs) > 0 {
		filter["projectId"] = bson.M{"$in": in.ProjectIDs}
	}
	if len(in.Tags) > 0 {
		filter["tags"] = bson.M{"$all": in.Tags}
	}
	if in.From != nil || in.To != nil {
		spentAt := bson.M{}
		if in.From != nil {
			spentAt["$gte"] = *in.From
		}
		if in.To != nil {
			spentAt["$lte"] = *in.To
		}
		filter["spentAt"] = spentAt
	}
	return filter
}

// Update applies fields to an expense scoped to orgID.
func (s *ExpenseStore) Update(ctx context.Context, orgID, id string, update dao.ExpenseUpdate) (entities.Expense, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.Expense{}, entities.ErrExpenseNotFound
	}

	set := bson.M{"updatedAt": update.UpdatedAt}
	if update.ProjectID != nil {
		set["projectId"] = *update.ProjectID
	}
	if update.Type != nil {
		set["type"] = *update.Type
	}
	if update.Amount != nil {
		set["amount"] = *update.Amount
	}
	if update.Description != nil {
		set["description"] = *update.Description
	}
	if update.Tags != nil {
		set["tags"] = *update.Tags
	}
	if update.SpentAt != nil {
		set["spentAt"] = *update.SpentAt
	}

	var document expenseDocument
	err = s.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID, "orgId": orgID},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entities.Expense{}, entities.ErrExpenseNotFound
	}
	if err != nil {
		return entities.Expense{}, fmt.Errorf("update expense: %w", err)
	}
	return document.expense(), nil
}

// Delete removes an expense scoped to orgID.
func (s *ExpenseStore) Delete(ctx context.Context, orgID, id string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return entities.ErrExpenseNotFound
	}

	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": objectID, "orgId": orgID})
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	if result.DeletedCount == 0 {
		return entities.ErrExpenseNotFound
	}
	return nil
}
