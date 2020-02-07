package pegnet

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/FactomWyomingEntity/prosper-pool/config"
	"github.com/FactomWyomingEntity/prosper-pool/database"
	"github.com/jinzhu/gorm"
	"github.com/pegnet/pegnet/modules/grader"
)

func InsertGradeBlock(tx *gorm.DB, eblock *factom.EBlock, graded grader.GradedBlock) error {
	next := database.PegnetGrade{
		Height:      int32(eblock.Height),
		Version:     graded.Version(),
		ShortHashes: strings.Join(graded.WinnersShortHashes(), ","),
		Cutoff:      graded.Cutoff(),
		Count:       graded.Count(),
		EblockKeyMr: eblock.KeyMR[:],
		PrevKeyMr:   eblock.PrevKeyMR[:],
		EbSequence:  int(eblock.Sequence),
	}

	return tx.FirstOrCreate(&next).Error
}

func (n *Node) Grade(ctx context.Context, block *factom.EBlock) (grader.GradedBlock, error) {
	if block == nil {
		// No block? Nothing to do
		return nil, nil
	}

	if bytes.Compare(block.ChainID[:], config.OPRChain[:]) != 0 {
		return nil, fmt.Errorf("trying to grade a non-opr chain")
	}

	ver := uint8(1)
	if block.Height >= config.GradingV2Activation {
		ver = 2
	}
	if block.Height >= config.FreeFloatingPEGPriceActivation {
		ver = 3
	}
	if block.Height >= config.V4OPRActivation {
		ver = 4
	}

	var prevWinners []string
	var prevGraded database.PegnetGrade
	dbErr := n.db.Order("height desc").
		Where("height < ?", block.Height).
		First(&prevGraded)
	if dbErr.Error == gorm.ErrRecordNotFound {
		// We have no prev winners, so the default is nil
		prevWinners = nil
	} else if dbErr.Error != nil {
		return nil, dbErr.Error
	} else {
		prevWinners = strings.Split(prevGraded.ShortHashes, ",")
	}

	g, err := grader.NewGrader(ver, int32(block.Height), prevWinners)
	if err != nil {
		return nil, err
	}

	for _, entry := range block.Entries {
		extids := make([][]byte, len(entry.ExtIDs))
		for i := range entry.ExtIDs {
			extids[i] = entry.ExtIDs[i]
		}
		// ignore bad opr errors
		err = g.AddOPR(entry.Hash[:], extids, entry.Content)
		if err != nil {
			// This is a noisy debug print
			// pegdLog.WithError(err).WithFields(log.Fields{"hash": entry.Hash.String()}).Debug("failed to add opr")
		}
	}

	return g.Grade(), nil
}
