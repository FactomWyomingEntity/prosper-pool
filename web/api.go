package web

import (
	"encoding/json"
	"net/http"

	"github.com/FactomWyomingEntity/prosper-pool/minutekeeper"

	"github.com/FactomWyomingEntity/prosper-pool/sharesubmit"

	"github.com/FactomWyomingEntity/prosper-pool/accounting"
	"github.com/FactomWyomingEntity/prosper-pool/database"
	rpc "github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json2"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

const (
	MaxLimit int32 = 200
)

func (s *HttpServices) APIMux(base string) *rpc.Server {
	apiMux := rpc.NewServer()
	apiMux.RegisterCodec(json2.NewCodec(), "application/json")
	err := apiMux.RegisterService(s, "api")
	if err != nil {
		log.WithError(err).Fatal("failed to create api")
	}

	return apiMux
}

type PoolBlockPerformanceResponse struct {
	Data       []accounting.OwedPayouts    `json:"data"`
	Pagination database.PaginationResponse `json:"info"`
}

// DB related apis
func (s *HttpServices) Rewards(r *http.Request, args *database.PaginationParams, reply *PoolBlockPerformanceResponse) error {
	args.Default(50, "desc", "job_id").Max(MaxLimit)
	db, err := database.SimplePagination(s.db, *args)
	if err != nil {
		return err
	}

	err = db.Find(&reply.Data).Error
	if err == gorm.ErrRecordNotFound {
		return nil // No records
	}
	if err != nil {
		return err
	}

	total := database.TotalCount(db.Model(&accounting.OwedPayouts{}))
	reply.Pagination.TotalRecords = total
	reply.Pagination.Records = len(reply.Data)
	return nil
}

type EntrySubmissionParams struct {
	JobID int32 `json:"jobid"`
	database.PaginationParams
}

type EntrySubmissionResponse struct {
	Data       []sharesubmit.PublicEntrySubmission `json:"data"`
	Pagination database.PaginationResponse         `json:"info"`
}

func (s *HttpServices) EntrySubmissions(r *http.Request, args *EntrySubmissionParams, reply *EntrySubmissionResponse) error {
	args.Default(50, "desc", "job_id").Max(MaxLimit)
	db, err := database.SimplePagination(s.db, args.PaginationParams)
	if err != nil {
		return err
	}

	// Filter
	if args.JobID != 0 {
		db = db.Where("job_id = ?", args.JobID)
	}

	err = db.Table("entry_submissions").Find(&reply.Data).Error
	if err == gorm.ErrRecordNotFound {
		return nil // No records
	}
	if err != nil {
		return err
	}

	total := database.TotalCount(db.Model(&sharesubmit.EntrySubmission{}))
	reply.Pagination.TotalRecords = total
	reply.Pagination.Records = len(reply.Data)
	return nil
}

func (s *HttpServices) SubmitSync(r *http.Request, _ *json.RawMessage, reply *minutekeeper.MinuteKeeperStatus) error {
	*reply = s.MinuteKeeper.Status()
	return nil
}
