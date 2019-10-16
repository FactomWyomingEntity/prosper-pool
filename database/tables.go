package database

import (
	"time"

	"github.com/jinzhu/gorm"
)

// PegnetGrade tracks all graded blocks for syncing state
type PegnetGrade struct {
	// Height of the graded block
	Height int32 `gorm:"primary_key"`

	Version uint8 // OPR Version
	// Winners of this block. If there are no winners, then the shorthashes
	// are the shorthashes of the previous block
	ShortHashes string

	// Cutoff is the the cutoff used in grading
	Cutoff int
	// Count is the number of oprs found for the height
	Count int

	// Some extra indexing and context that we might want
	EblockKeyMr []byte
	PrevKeyMr   []byte
	EbSequence  int
}

// PegnetPayout tracks all PEG rewards
type PegnetPayout struct {
	ID       uint `gorm:"primary_key"`
	Height   int32
	Position int32
	Reward   int64
	// create index with name `addr` for address
	CoinbaseAddress string `gorm:"index:addr"`
	Identity        string
	EntryHash       []byte
}

// BlockSync indicates the latest block height that has been fully synced.
type BlockSync struct {
	Synced     int32 `gorm:"primary_key"`
	SyncedDate time.Time
}

func (sync *BlockSync) BeforeCreate(scope *gorm.Scope) error {
	sync.SyncedDate = time.Now()
	return nil
}
