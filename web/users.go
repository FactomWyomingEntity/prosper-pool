package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/FactomWyomingEntity/prosper-pool/accounting"
	"github.com/FactomWyomingEntity/prosper-pool/authentication"
	"github.com/FactomWyomingEntity/prosper-pool/sharesubmit"
)

func (s *HttpServices) Nav() []byte {
	return []byte(`<a href="/users">Users</a> ` +
		`<a href="/pool">Pool</a> ` +
		`<a href="/admin/links">Admin</a><br />`)
}

func (s *HttpServices) Index(w http.ResponseWriter, r *http.Request) {
	w.Write(s.Nav())
}

func (s *HttpServices) AdminLinks(w http.ResponseWriter, r *http.Request) {
	w.Write(s.Nav())

	w.Write([]byte(`
	<ul>
		<li><a href="/admin/miners">Miners</a></li>
	</ul>
	`))
}

func (s *HttpServices) UserLinks(w http.ResponseWriter, r *http.Request) {
	w.Write(s.Nav())

	w.Write([]byte(`
	<ul>
		<li><a href="/whoami">WhoAmI?</a></li>
		<li><a href="/user/owed">Owed</a></li>
		<li><a href="/auth/login">Login</a></li>
		<li><a href="/auth/logout">Logout</a></li>
	</ul>
	`))
}

func (s *HttpServices) PoolLinks(w http.ResponseWriter, r *http.Request) {
	w.Write(s.Nav())

	w.Write([]byte(`
	<ul>
		<li><a href="/pool/submissions">Submissions</a></li>
		<li><a href="/pool/rewards">Rewards</a></li>
	</ul>
	`))
}

func (s *HttpServices) WhoAmI(w http.ResponseWriter, r *http.Request) {
	w.Write(s.Nav())
	w.Write([]byte("<pre>"))
	defer w.Write([]byte("</pre>"))
	user, err := s.GetCurrentUser(r)
	if err != nil {
		_, _ = fmt.Fprintf(w, "Error:%s", err.Error())
		return
	}
	_, _ = fmt.Fprintf(w, "Hello %s", user.UID)
}

func (s *HttpServices) GetCurrentUser(r *http.Request) (*authentication.User, error) {
	user := s.Auth.GetCurrentUser(r)
	if user != nil {
		if uc, ok := user.(*authentication.User); ok {
			return uc, nil
		}
		return nil, fmt.Errorf("internal error: unknown user")
	}
	return nil, fmt.Errorf("not logged in")
}

func (s *HttpServices) OwedPayouts(w http.ResponseWriter, r *http.Request) {
	w.Write(s.Nav())
	w.Write([]byte("<pre>"))
	defer w.Write([]byte("</pre>"))

	user, err := s.GetCurrentUser(r)
	if err != nil {
		_, _ = fmt.Fprintf(w, "Error:%s", err.Error())
		return
	}

	// Only grab last 100 blocks
	var ious []accounting.UserOwedPayouts
	s.db.Order("job_id desc").Where("user_id = ?", user.UID).Limit(100).Find(&ious)

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("This page displays the last 100 owed payouts for %s\n", user.UID))
	for _, iou := range ious {
		buf.WriteString(fmt.Sprintf("\tHeight: %d, PEG: %s, Proportion: %s, Shares: %.2f, HashRate: %.2f h\\s\n",
			iou.JobID, FactoshiToFactoid(uint64(iou.Payout)),
			iou.Proportion.Truncate(3).String(), iou.UserDifficuty,
			iou.HashRate))
	}
	_, _ = w.Write(buf.Bytes())
}

func (s *HttpServices) PoolRewards(w http.ResponseWriter, r *http.Request) {
	w.Write(s.Nav())
	w.Write([]byte("<pre>"))
	defer w.Write([]byte("</pre>"))
	// Only grab last 100 blocks
	var rewards []accounting.OwedPayouts
	s.db.Order("job_id desc").Limit(100).Find(&rewards)

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("This page displays the last 100 pool rewards\n"))
	for _, rew := range rewards {
		buf.WriteString(fmt.Sprintf("\tHeight: %d, PEG: %s, Difficulty: %.2f, HashRate: %.2f h\\s\n",
			rew.JobID, FactoshiToFactoid(uint64(rew.PoolReward)),
			rew.PoolDifficuty, rew.TotalHashrate))
	}
	_, _ = w.Write(buf.Bytes())
}

func (s *HttpServices) PoolSubmissions(w http.ResponseWriter, r *http.Request) {
	w.Write(s.Nav())
	w.Write([]byte("<pre>"))
	defer w.Write([]byte("</pre>"))
	jobid := r.FormValue("jobid")
	if jobid == "" {
		_, _ = w.Write([]byte("no jobid provided"))
		return
	}

	var entries []sharesubmit.EntrySubmission

	// TODO: Verify no sql injection possibility
	dbErr := s.db.Model(&sharesubmit.EntrySubmission{}).
		Where("job_id = ?", jobid).
		Order("created_at desc").
		Find(&entries)
	if dbErr.Error != nil {
		_, _ = w.Write([]byte(dbErr.Error.Error()))
		return
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("This page displays the %d entry submissions for the job '%s'\n", len(entries), jobid))
	buf.WriteString(fmt.Sprintf("All 0 entryhashes means the submission was blocked by softmax.\n"))
	blocked := 0
	for _, entry := range entries {
		if entry.EntryHash == "0000000000000000000000000000000000000000000000000000000000000000" {
			blocked++
		}
	}
	buf.WriteString(fmt.Sprintf("%d out of %d blocked by softmax (%.3f%% of submissions blocked)\n",
		blocked, len(entries), 100*float64(blocked)/float64(len(entries))))
	for i, entry := range entries {
		buf.WriteString(fmt.Sprintf("\t%d -> EntryHash: %s, Target: %x, Time: %s\n",
			i, entry.EntryHash, entry.Target, entry.CreatedAt.UTC()))
	}

	_, _ = w.Write(buf.Bytes())
}

func (s *HttpServices) PoolMiners(w http.ResponseWriter, r *http.Request) {
	w.Write(s.Nav())
	w.Write([]byte("<pre>"))
	defer w.Write([]byte("</pre>"))
	// TODO: Add auth protection
	if s.StratumServer == nil {
		_, _ = w.Write([]byte("No stratum server hooked up"))
		return
	}

	miners := s.StratumServer.MinersSnapShot()
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("This page displays all connected miners\n"))
	buf.WriteString(fmt.Sprintf("  %d total miners\n", len(miners)))
	for i, miner := range miners {
		buf.WriteString(fmt.Sprintf("-------- %d:  Miner %s/%s --------\n", i, miner.Username, miner.Minerid))
		buf.WriteString(fmt.Sprintf("\t%10s: %s\n", "Agent", miner.Agent))
		buf.WriteString(fmt.Sprintf("\t%10s: %s\n", "IP", miner.IP))
		buf.WriteString(fmt.Sprintf("\t%10s: %s\n", "Session", miner.SessionID))
		buf.WriteString(fmt.Sprintf("\t%10s: %t\n", "Auth", miner.Authorized))
		buf.WriteString(fmt.Sprintf("\t%10s: %t\n", "Sub", miner.Subscribed))
		buf.WriteString(fmt.Sprintf("\t%10s: %x\n", "PrefTarget", miner.PrefferedTarget))
		buf.WriteString(fmt.Sprintf("\t%10s: %d\n", "Nonce", miner.Nonce))
	}
	_, _ = w.Write(buf.Bytes())
}

// MinuteKeeperInfo has the json endpoint to indicate if submissions are being
// accepted.
func (s *HttpServices) MinuteKeeperInfo(w http.ResponseWriter, r *http.Request) {
	if s.MinuteKeeper == nil {
		_, _ = w.Write([]byte(`{"error":"no minute keeper hook up"}`))
		return
	}

	info, _ := json.Marshal(s.MinuteKeeper.Status())
	_, _ = w.Write(info)
}

// FactoshiToFactoid converts a uint64 factoshi ammount into a fixed point
// number represented as a string
func FactoshiToFactoid(i uint64) string {
	d := i / 1e8
	r := i % 1e8
	ds := fmt.Sprintf("%d", d)
	rs := fmt.Sprintf("%08d", r)
	rs = strings.TrimRight(rs, "0")
	if len(rs) > 0 {
		ds = ds + "."
	}
	return fmt.Sprintf("%s%s", ds, rs)
}

// FactoidToFactoshi takes a Factoid amount as a string and returns the value in
// factoids
func FactoidToFactoshi(amt string) uint64 {
	valid := regexp.MustCompile(`^([0-9]+)?(\.[0-9]+)?$`)
	if !valid.MatchString(amt) {
		return 0
	}

	var total uint64 = 0

	dot := regexp.MustCompile(`\.`)
	pieces := dot.Split(amt, 2)
	whole, _ := strconv.Atoi(pieces[0])
	total += uint64(whole) * 1e8

	if len(pieces) > 1 {
		a := regexp.MustCompile(`(0*)([0-9]+)$`)

		as := a.FindStringSubmatch(pieces[1])
		part, _ := strconv.Atoi(as[0])
		power := len(as[1]) + len(as[2])
		total += uint64(part * 1e8 / int(math.Pow10(power)))
	}

	return total
}
