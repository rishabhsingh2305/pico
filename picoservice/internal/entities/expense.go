package entities

import "time"

// ExpenseType describes the direction of an expense. Debit means money left
// the account and credit means money came in.
type ExpenseType string

const (
	ExpenseTypeDebit  ExpenseType = "debit"
	ExpenseTypeCredit ExpenseType = "credit"
)

// OrDefault returns debit for legacy records that do not have a type.
func (t ExpenseType) OrDefault() ExpenseType {
	if t == "" {
		return ExpenseTypeDebit
	}
	return t
}

// Valid reports whether t is a supported expense type.
func (t ExpenseType) Valid() bool {
	return t == ExpenseTypeDebit || t == ExpenseTypeCredit
}

// Expense is an expense record.
type Expense struct {
	ID          string
	OrgID       string
	UserID      string
	ProjectID   string
	Type        ExpenseType
	Amount      int64
	Currency    string
	Description string
	Tags        []string
	SpentAt     time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
