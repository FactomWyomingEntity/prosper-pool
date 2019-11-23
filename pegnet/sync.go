package pegnet

import (
	"context"
	"time"

	"github.com/pegnet/pegnet/modules/grader"

	"github.com/FactomWyomingEntity/prosper-pool/database"

	"github.com/jinzhu/gorm"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/FactomWyomingEntity/prosper-pool/config"
	log "github.com/sirupsen/logrus"
)

// DBlockSync syncs the blockchain block by block. We can improve our sync a bit
// by walking through only the eblocks of the oprs, but that increasing
// complexity by having to maintain the eblock chain. If we sync by heights
// we are guaranteed to always sync in order. And most blocks are filled
// with oprs anyway, so eblock syncing doesn't buy much.
func (n *Node) DBlockSync(ctx context.Context) {
	n.justBooted = true
	pollingPeriod := n.config.GetDuration(config.ConfigPegnetPollingPeriod)
	retryPeriod := n.config.GetDuration(config.ConfigPegnetRetryPeriod)

OuterSyncLoop:
	for {
		if ctx.Err() != nil {
			return // ctx is cancelled
		}

		// Fetch the current highest height
		heights := new(factom.Heights)

		// TODO: we might want to query against more than 1 factomd. If 1 node
		// 	is synced higher than the other (such as following minutes better)
		//	we will want to switch the client.
		err := heights.Get(nil, n.FactomClient)
		if err != nil {
			pegdLog.WithError(err).WithFields(log.Fields{"fhost": n.FactomClient.FactomdServer}).Errorf("failed to fetch heights")
			time.Sleep(retryPeriod)
			continue // Loop will just keep retrying until factomd is reached
		}

		if n.Sync.Synced >= int32(heights.DirectoryBlock) {
			// We are currently synced, nothing to do. If we are above it, the factomd could
			// be rebooted
			// TODO: Reduce polling period depending on what minute we are in

			if n.Sync.Synced == int32(heights.DirectoryBlock) && n.justBooted {
				// We want to send the last job down
				n.Sync.Synced--
			} else {
				time.Sleep(pollingPeriod)
				continue
			}
		}

		var totalDur time.Duration
		var iterations int

		n.justBooted = false
		begin := time.Now()
		for n.Sync.Synced < int32(heights.DirectoryBlock) {
			current := n.Sync.Synced + 1
			start := time.Now()
			hLog := pegdLog.WithFields(log.Fields{"height": current, "dheight": heights.DirectoryBlock, "hooks": len(n.hooks)})
			if ctx.Err() != nil {
				return // ctx is cancelled
			}

			// start transaction for all block actions
			tx := n.db.BeginTx(ctx, nil)
			if tx.Error != nil {
				hLog.WithError(err).Errorf("failed to start transaction")
				time.Sleep(retryPeriod)
				continue OuterSyncLoop
			}

			// We are not synced, so we need to iterate through the dblocks and sync them
			// one by one. We can only sync our current synced height +1
			// TODO: This skips the genesis block. I'm sure that is fine
			block, err := n.SyncBlock(ctx, tx, uint32(current))
			if err != nil {
				hLog.WithError(err).Errorf("failed to sync height")
				// If we fail, we backout to the outer loop. This allows error handling on factomd state to be a bit
				// cleaner, such as a rebooted node with a different db. That node would have a new heights response.
				dbErr := tx.Rollback()
				if dbErr.Error != nil {
					hLog.WithError(err).Fatal("unable to roll back transaction")
				}
				time.Sleep(retryPeriod)
				continue OuterSyncLoop
			}

			// Bump our sync, and march forward

			n.Sync.Synced++

			dbErr := tx.FirstOrCreate(n.Sync)
			if dbErr.Error != nil {
				n.Sync.Synced--
				hLog.WithError(err).Errorf("unable to update synced metadata")
				dbErr = tx.Rollback()
				if dbErr.Error != nil {
					hLog.WithError(dbErr.Error).Fatal("unable to roll back transaction")
				}
				time.Sleep(retryPeriod)
				continue OuterSyncLoop
			}

			dbErr = tx.Commit()
			if dbErr.Error != nil {
				n.Sync.Synced--
				hLog.WithError(dbErr.Error).Errorf("unable to commit transaction")
				dbErr = tx.Rollback()
				if dbErr.Error != nil {
					hLog.WithError(dbErr.Error).Fatal("unable to roll back transaction")
				}
				time.Sleep(retryPeriod)
				continue OuterSyncLoop
			}

			elapsed := time.Since(start)

			hLog.WithFields(log.Fields{"took": elapsed}).Debugf("synced")

			// TODO: Insert hook for mining
			// TODO: Eval efficiency of this sync.
			pegnetSyncHeight.Set(float64(n.Sync.Synced))

			// Send the new block to anyone listening
			// TODO: Ensure this logic is correct.
			hook := PegnetdHook{
				GradedBlock: block,
				Top:         current == int32(heights.DirectoryBlock),
				Height:      current,
			}
			// Don't bother nil blocks
			if hook.GradedBlock != nil {
				for i := range n.hooks {
					select {
					case n.hooks[i] <- hook:
						hLog.Tracef("hook sent")
					default:
						hLog.Warnf("hook failed to send")
					}
				}
			}

			iterations++
			totalDur += elapsed
			// Only print if we are > 50 behind and every 50
			if iterations%50 == 0 {
				toGo := int32(heights.DirectoryBlock) - n.Sync.Synced
				avg := totalDur / time.Duration(iterations)
				hLog.WithFields(log.Fields{
					"avg":        avg,
					"left":       time.Duration(toGo) * avg,
					"syncing-to": heights.DirectoryBlock,
					"elapsed":    time.Since(begin),
				}).Infof("sync stats")
			}
		}
	}
}

// If SyncBlock returns no error, than that height was synced and saved. If any
// part of the sync fails, the whole sync should be rolled back and not applied.
// An error should then be returned. The context should be respected if it is
// cancelled
func (n *Node) SyncBlock(ctx context.Context, tx *gorm.DB, height uint32) (grader.GradedBlock, error) {
	fLog := pegdLog.WithFields(log.Fields{"height": height})
	if err := ctx.Err(); err != nil { // Just an example about how to handle it being cancelled
		return nil, err
	}

	dblock := new(factom.DBlock)
	dblock.Height = height
	if err := dblock.Get(nil, n.FactomClient); err != nil {
		return nil, err
	}

	// First, gather all entries we need from factomd
	oprEBlock := dblock.EBlock(factom.Bytes32(config.OPRChain))
	if oprEBlock != nil {
		if err := multiFetch(oprEBlock, n.FactomClient); err != nil {
			return nil, err
		}
	}

	// Then, grade the new OPR Block. The results of this will be used
	// to execute conversions that are in holding.
	gradedBlock, err := n.Grade(ctx, oprEBlock)
	if err != nil {
		return nil, err
	} else if gradedBlock != nil {
		err = InsertGradeBlock(tx, oprEBlock, gradedBlock)
		if err != nil {
			return nil, err
		}
		winners := gradedBlock.Winners()
		if 0 < len(winners) {
			var s database.PegnetPayout
			err := tx.Order("height desc").First(&s).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return nil, err
			}
			if s.Height != int32(height) {
				// Write the top 50, not just the top 25
				graded := gradedBlock.Graded()
				for i := range graded {
					payout := database.PegnetPayout{
						Height:          int32(height),
						Position:        int32(graded[i].Position()),
						Reward:          int64(graded[i].Payout()),
						CoinbaseAddress: graded[i].OPR.GetAddress(),
						Identity:        graded[i].OPR.GetID(),
						EntryHash:       graded[i].EntryHash,
					}
					if dbErr := tx.Create(&payout); dbErr.Error != nil {
						return nil, dbErr.Error
					}
				}
			}
		} else {
			fLog.WithFields(log.Fields{"section": "grading", "reason": "no winners"}).Tracef("block not graded")
		}
	} else {
		fLog.WithFields(log.Fields{"section": "grading", "reason": "no graded block"}).Tracef("block not graded")
	}

	return gradedBlock, nil
}

func multiFetch(eblock *factom.EBlock, c *factom.Client) error {
	err := eblock.Get(nil, c)
	if err != nil {
		return err
	}

	work := make(chan int, len(eblock.Entries))
	defer close(work)
	errs := make(chan error)
	defer close(errs)

	for i := 0; i < 8; i++ {
		go func() {
			// TODO: Fix the channels such that a write on a closed channel never happens.
			//		For now, just kill the worker go routine
			defer func() {
				recover()
			}()

			for j := range work {
				errs <- eblock.Entries[j].Get(nil, c)
			}
		}()
	}

	for i := range eblock.Entries {
		work <- i
	}

	count := 0
	for e := range errs {
		count++
		if e != nil {
			// If we return, we close the errs channel, and the working go routine will
			// still try to write to it.
			return e
		}
		if count == len(eblock.Entries) {
			break
		}
	}

	return nil
}
