package export

import (
	"encoding/csv"
	"io"
	"time"

	"github.com/viniciushammett/go-access-auditor/internal/store"
)

func WriteCSV(w io.Writer, events []store.Event) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id","when","user","host","source","command","sensitive","rule"})
	for _, e := range events {
		_ = cw.Write([]string{
			e.ID, e.When.Format(time.RFC3339), e.User, e.Host, e.Source, e.Command,
			boolStr(e.FlagSensitive), e.RuleMatched,
		})
	}
	cw.Flush()
	return cw.Error()
}
func boolStr(b bool) string { if b { return "true" }; return "false" }