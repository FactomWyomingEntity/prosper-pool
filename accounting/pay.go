package accounting

import (
	"database/sql"
	"fmt"

	"github.com/FactomWyomingEntity/private-pool/authentication"
	"github.com/jinzhu/gorm"
)

type Paid struct {
	gorm.Model    `json:"-"`
	EntryHash     string
	UserID        string `gorm:"index:user_id"`
	PayoutAddress string
	PaymentAmount int64

	// tmp fields for debugging
	TotalOwed int64 `gorm:"-"`
	TotalPaid int64 `gorm:"-"`
}

// CalculatePayments does not insert the payments. It just preps them for
// insert
func (a *Accountant) CalculatePayments() ([]Paid, error) {
	var users []authentication.User
	err := a.DB.Find(&users).Error
	if err != nil {
		return nil, err
	}

	// Entryhash will not be filled out, since we don't know it yet
	payments := make([]Paid, len(users))

	for i, u := range users {
		payments[i].UserID = u.UID
		payments[i].PayoutAddress = u.PayoutAddress
		// Sum up what we paid
		var paid sql.NullInt64
		paidRow := a.DB.Table("paids").
			Where("user_id = ?", u.UID).Select("sum(payment_amount)").Row()
		err = paidRow.Scan(&paid)
		if err != nil {
			return nil, err
		}
		payments[i].TotalPaid = paid.Int64

		// Sum up what we owe
		var owed sql.NullInt64
		owedRow := a.DB.Table("user_owed_payouts").
			Where("user_id = ?", u.UID).Select("sum(payout)").Row()
		err = owedRow.Scan(&owed)
		if err != nil {
			return nil, err
		}
		payments[i].TotalOwed = owed.Int64

		payments[i].PaymentAmount = payments[i].TotalOwed - payments[i].TotalPaid
	}

	return payments, nil
}

func (a *Accountant) WritePayments(payments []Paid) error {
	if len(payments) == 0 {
		return fmt.Errorf("no payments to record")
	}

	if payments[0].EntryHash == "" {
		return fmt.Errorf("this is not a receipt, no entryhash")
	}
	var f Paid
	res := a.DB.Model(&Paid{}).Where("entry_hash = ?", payments[0].EntryHash).First(&f)
	if res.RowsAffected > 0 {
		return fmt.Errorf("this tx is already recorded")
	}

	tx := a.DB.Begin()
	for _, payment := range payments {
		err := tx.Create(&payment).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}
